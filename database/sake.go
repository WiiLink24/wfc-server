package database

import (
	"context"
	"encoding/json"
	"errors"
	"wwfc/filter"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	SakeFieldTypeByte          = 0
	SakeFieldTypeShort         = 1
	SakeFieldTypeInt           = 2
	SakeFieldTypeFloat         = 3
	SakeFieldTypeAsciiString   = 4
	SakeFieldTypeUnicodeString = 5
	SakeFieldTypeBoolean       = 6
	SakeFieldTypeDateAndTime   = 7
	SakeFieldTypeBinaryData    = 8
	SakeFieldTypeInt64         = 9
)

type SakeFieldType int

type SakeField struct {
	Type  SakeFieldType `json:"type"`
	Value string        `json:"value"`
}

type SakeRecord struct {
	GameId   int
	OwnerId  int32
	TableId  string
	RecordId int32
	Fields   map[string]SakeField
}

const (
	getSakeRecordsQuery = `
		SELECT owner_id, record_id, fields
		FROM sake_records
		WHERE game_id = $1
		  AND table_id = $2
		  AND (cardinality($4::integer[]) = 0 OR record_id = ANY($4::integer[]))
		  AND (cardinality($3::integer[]) = 0 OR owner_id = ANY($3::integer[]))`

	updateSakeRecordQuery = `
        UPDATE sake_records 
        SET 
            fields = CASE WHEN owner_id = $4 THEN fields || $5 ELSE fields END,
            update_time = CASE WHEN owner_id = $4 THEN CURRENT_TIMESTAMP ELSE update_time END
        WHERE game_id = $1 
          AND table_id = $2 
          AND record_id = $3
        RETURNING owner_id`

	insertSakeRecordQuery = `
		INSERT INTO sake_records (game_id, table_id, owner_id, fields) 
		VALUES ($1, $2, $3, $4)
		RETURNING record_id`
)

var (
	ErrSakeNotOwned = errors.New("record is not owned by the specified owner ID")
)

func parseSakeFieldsFromJson(fieldsJson []byte) (map[string]SakeField, error) {
	var fields map[string]SakeField
	err := json.Unmarshal(fieldsJson, &fields)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

func GetSakeRecords(pool *pgxpool.Pool, ctx context.Context, gameId int, ownerIds []int32, tableId string, recordIds []int32, fields []string, filterExpr string) ([]SakeRecord, error) {
	if fields == nil {
		fields = []string{}
	}
	if ownerIds == nil {
		ownerIds = []int32{}
	}
	if recordIds == nil {
		recordIds = []int32{}
	}

	query := getSakeRecordsQuery
	if filterExpr != "" {
		tree, err := filter.Parse(filterExpr)
		if err != nil {
			return nil, err
		}

		var filterQuery string
		err = pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
			filterQuery, err = createSqlFilter(conn.Conn().PgConn(), tree)
			return err
		})
		if err != nil {
			return nil, err
		}

		// This filter has been entirely rewritten by our filter code,
		// based on the expression supplied by the user. This should be safe!!!
		query += " AND (" + filterQuery + ")"
	}

	rows, err := pool.Query(ctx, query, gameId, tableId, ownerIds, recordIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SakeRecord
	for rows.Next() {
		record := SakeRecord{
			GameId:  gameId,
			TableId: tableId,
		}
		var fieldsJson []byte
		if err := rows.Scan(&record.OwnerId, &record.RecordId, &fieldsJson); err != nil {
			return nil, err
		}
		fields, err := parseSakeFieldsFromJson(fieldsJson)
		if err != nil {
			return nil, err
		}
		record.Fields = fields

		records = append(records, record)
	}

	return records, nil
}

func UpdateSakeRecord(pool *pgxpool.Pool, ctx context.Context, record SakeRecord, ownerId int32) error {
	fieldsJson, err := json.Marshal(record.Fields)
	if err != nil {
		return err
	}
	var existingOwnerId int32
	err = pool.QueryRow(ctx, updateSakeRecordQuery, record.GameId, record.TableId, record.RecordId, ownerId, fieldsJson).Scan(&existingOwnerId)
	if err != nil {
		return err
	}
	if ownerId != 0 && existingOwnerId != ownerId {
		return ErrSakeNotOwned
	}

	return nil
}

func InsertSakeRecord(pool *pgxpool.Pool, ctx context.Context, record SakeRecord) (recordId int32, err error) {
	fieldsJson, err := json.Marshal(record.Fields)
	if err != nil {
		return 0, err
	}

	for i := 0; i < 10; i++ {
		err = pool.QueryRow(ctx, insertSakeRecordQuery, record.GameId, record.TableId, record.OwnerId, fieldsJson).Scan(&recordId)
		if err == nil {
			break
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			break
		}
		if pgErr.Code != pgerrcode.UniqueViolation {
			break
		}
		// Retry if unique violation occurred, as the record ID is generated randomly
	}
	return recordId, err
}
