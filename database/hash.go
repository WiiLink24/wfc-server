package database

import (
	"context"
	"errors"
	"regexp"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"github.com/sasha-s/go-deadlock"
)

const (
	QueryHashes = `SELECT * FROM hashes`
	InsertHash  = `INSERT
		INTO hashes (pack_id, version, hash_ntscu, hash_ntscj, hash_ntsck, hash_pal)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pack_id, version)
		DO UPDATE SET hash_ntscu = $3, hash_ntscj = $4, hash_ntsck = $5, hash_pal = $6`
	DeleteHash = `DELETE FROM hashes WHERE pack_id = $1 AND version = $2`
)

type Region byte

const (
	R_NTSCU Region = iota
	R_NTSCJ
	R_NTSCK
	R_PAL
)

func IndexByRegionByte(hbr HashesByRegion, b Region) string {
	switch b {
	case R_NTSCU:
		return hbr.NTSCU
	case R_NTSCJ:
		return hbr.NTSCJ
	case R_NTSCK:
		return hbr.NTSCK
	case R_PAL:
		return hbr.PAL
	default:
		return ""
	}
}

type HashesByRegion struct {
	NTSCU string
	NTSCJ string
	NTSCK string
	PAL   string
}

type HashStore map[uint32]map[uint32]HashesByRegion

var (
	mutex       = deadlock.Mutex{}
	hashes      = HashStore{}
	emptyRegexp *regexp.Regexp
)

// Used to flatten the 40-wide empty strings returned by the db, and to filter
// whitespace potentially submitted over the api
func flattenBlank(str string) string {
	if emptyRegexp.MatchString(str) {
		return ""
	}

	return str
}

func HashInit(pool *pgxpool.Pool, ctx context.Context) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Populate regexp once
	emptyRegexp = regexp.MustCompile("^\\s+$")

	logging.Info("DATABASE", "Populating hashes from the database")

	rows, err := pool.Query(ctx, QueryHashes)
	defer rows.Close()
	if err != nil {
		return err
	}

	for rows.Next() {
		var packID uint32
		var version uint32
		var hashNTSCU string
		var hashNTSCJ string
		var hashNTSCK string
		var hashPAL string

		err = rows.Scan(&packID, &version, &hashNTSCU, &hashNTSCJ, &hashNTSCK, &hashPAL)
		if err != nil {
			return err
		}

		versions, exists := hashes[packID]

		if !exists {
			temp := map[uint32]HashesByRegion{}
			hashes[packID] = temp
			versions = temp
		}

		versions[version] = HashesByRegion{
			NTSCU: flattenBlank(hashNTSCU),
			NTSCJ: flattenBlank(hashNTSCJ),
			NTSCK: flattenBlank(hashNTSCK),
			PAL:   flattenBlank(hashPAL),
		}

		logging.Info("DATABASE", "Populated hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version), "\nNTSCU:", aurora.Cyan(hashNTSCU), "\nNTSCJ:", aurora.Cyan(hashNTSCJ), "\nNTSCK:", aurora.Cyan(hashNTSCK), "\nPAL:", aurora.Cyan(hashPAL))
	}

	return nil
}

func GetHashes() HashStore {
	mutex.Lock()
	defer mutex.Unlock()

	// Create a copy while the mutex is locked. Defer runs before return
	ret := hashes

	return ret
}

var (
	ErrPackIDMissing  = errors.New("The specified PackID does not exist")
	ErrVersionMissing = errors.New("The specified version does not exist")
)

func RemoveHash(pool *pgxpool.Pool, ctx context.Context, packID uint32, version uint32) error {
	mutex.Lock()
	defer mutex.Unlock()

	if versions, exists := hashes[packID]; exists {
		if _, exists := versions[version]; exists {
			delete(versions, version)
		} else {
			return ErrVersionMissing
		}
	} else {
		return ErrPackIDMissing
	}

	_, err := pool.Exec(ctx, DeleteHash, packID, version)
	if err != nil {
		logging.Error("DATABASE", "Failure to remove hash for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version), "Error:", err.Error())
	} else {
		logging.Warn("DATABASE", "Removed hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version))
	}
	return err
}

func UpdateHash(pool *pgxpool.Pool, ctx context.Context, packID uint32, version uint32, hashNTSCU string, hashNTSCJ string, hashNTSCK string, hashPAL string) error {
	mutex.Lock()
	defer mutex.Unlock()

	versions, exists := hashes[packID]

	if !exists {
		temp := map[uint32]HashesByRegion{}
		hashes[packID] = temp
		versions = temp
	}

	versions[version] = HashesByRegion{
		NTSCU: flattenBlank(hashNTSCU),
		NTSCJ: flattenBlank(hashNTSCJ),
		NTSCK: flattenBlank(hashNTSCK),
		PAL:   flattenBlank(hashPAL),
	}

	_, err := pool.Exec(ctx, InsertHash, packID, version, hashNTSCU, hashNTSCJ, hashNTSCK, hashPAL)

	if err != nil {
		logging.Error("DATABASE", "Failed to update hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version), "error:", err.Error())
	} else {
		logging.Info("DATABASE", "Successfully updated hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version))
	}

	return err
}

func ValidateHash(packID uint32, version uint32, region Region, hash string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if versions, exists := hashes[packID]; exists {
		if regions, exists := versions[version]; exists {
			hash_real := IndexByRegionByte(regions, region)

			return hash_real != "" && hash_real == hash
		}
	}

	return false
}
