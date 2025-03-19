package database

import (
	"context"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"github.com/sasha-s/go-deadlock"
)

const (
	GetHashes  = `SELECT * FROM hashes`
	InsertHash = `INSERT
		INTO hashes (pack_id, version, hash_ntscu, hash_ntscj, hash_ntsck, hash_pal)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pack_id, version)
		DO UPDATE SET hash_ntscu = $3, hash_ntscj = $4, hash_ntsck = $5, hash_pal = $6`
)

type Region byte

const (
	R_NTSCU Region = iota
	R_NTSCJ
	R_NTSCK
	R_PAL
)

var (
	mutex  = deadlock.Mutex{}
	hashes = map[uint32]map[uint32]map[Region]string{}
)

func HashInit(pool *pgxpool.Pool, ctx context.Context) error {
	mutex.Lock()
	defer mutex.Unlock()

	logging.Info("DB", "Populating hashes from the database")

	rows, err := pool.Query(ctx, GetHashes)
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
			temp := map[uint32]map[Region]string{}
			hashes[packID] = temp
			versions = temp
		}

		regions, exists := versions[version]

		if !exists {
			temp := map[Region]string{}
			versions[version] = temp
			regions = temp
		}

		regions[R_NTSCU] = hashNTSCU
		regions[R_NTSCJ] = hashNTSCJ
		regions[R_NTSCK] = hashNTSCK
		regions[R_PAL] = hashPAL

		logging.Info("DB", "Populated hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version), "\nNTSCU:", aurora.Cyan(hashNTSCU), "\nNTSCJ:", aurora.Cyan(hashNTSCJ), "\nNTSCK:", aurora.Cyan(hashNTSCK), "\nPAL:", aurora.Cyan(hashPAL))
	}

	return nil
}

func UpdateHash(pool *pgxpool.Pool, ctx context.Context, packID uint32, version uint32, hashNTSCU string, hashNTSCJ string, hashNTSCK string, hashPAL string) error {
	mutex.Lock()
	defer mutex.Unlock()

	versions, exists := hashes[packID]

	if !exists {
		temp := map[uint32]map[Region]string{}
		hashes[packID] = temp
		versions = temp
	}

	regions, exists := versions[packID]

	if !exists {
		temp := map[Region]string{}
		versions[packID] = temp
		regions = temp
	}

	regions[R_NTSCU] = hashNTSCU
	regions[R_NTSCJ] = hashNTSCJ
	regions[R_NTSCK] = hashNTSCK
	regions[R_PAL] = hashPAL

	_, err := pool.Exec(ctx, InsertHash, packID, version, hashNTSCU, hashNTSCJ, hashNTSCK, hashPAL)

	if err != nil {
		logging.Error("DB", "Failed to update hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version), "error:", err.Error())
	} else {
		logging.Info("DB", "Successfully updated hashes for PackID:", aurora.Cyan(packID), "Version:", aurora.Cyan(version))
	}

	return err
}

func ValidateHash(packID uint32, version uint32, region Region, hash string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if versions, exists := hashes[packID]; exists {
		if regions, exists := versions[version]; exists {
			if hash_real, exists := regions[region]; exists {
				return hash_real != "" && hash_real == hash
			}
		}
	}

	return false
}
