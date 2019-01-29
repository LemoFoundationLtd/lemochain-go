package store

import (
	"database/sql"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

type DBWriter interface {
	SetIndex(flg int, route []byte, key []byte, offset int64) error
}

type DBReader interface {
	GetIndex(key []byte) (int, []byte, int64, error)
}

type DB interface {
	DBWriter
	DBReader
}

type MySqlDB struct {
	engine *sql.DB
	driver string
	dns    string
}

func NewMySqlDB(driver string, dns string) *MySqlDB {
	db, err := Open(driver, dns)
	if err != nil {
		panic("OPEN MYSQL DATABASE ERROR." + err.Error())
	}

	return &MySqlDB{
		engine: db,
		driver: driver,
		dns:    dns,
	}
}

func (db *MySqlDB) SetIndex(flg int, route []byte, key []byte, offset int64) error {
	tmp := common.ToHex(key[:])
	_, err := db.engine.Exec("REPLACE INTO t_kv(lm_flg, lm_key, lm_val, lm_pos) VALUES (?,?,?,?)", flg, tmp, route, offset)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *MySqlDB) GetIndex(key []byte) (int, []byte, int64, error) {
	tmp := common.ToHex(key[:])
	row := db.engine.QueryRow("SELECT lm_flg, lm_val, lm_pos FROM t_kv WHERE lm_key = ?", tmp)
	var flg int
	var val []byte
	var pos int64
	err := row.Scan(&flg, &val, &pos)
	if err == sql.ErrNoRows {
		return -1, nil, -1, nil
	}
	if err != nil {
		return -1, nil, -1, err
	}

	return flg, val, pos, nil
}

var txsql string = "SELECT tx_block_key, tx_val, tx_ver, tx_st FROM t_tx"

func (db *MySqlDB) TxSet(hash, blockHash, from, to string, val []byte, ver int64, st int64) error {
	_, err := db.engine.Exec("REPLACE INTO t_tx(tx_key, tx_block_key, tx_from, tx_to, tx_val, tx_ver, tx_st) VALUES (?,?,?,?,?,?,?)", hash, blockHash, from, to, val, ver, st)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *MySqlDB) TxGetByHash(key string) (string, []byte, int64, error) {
	row := db.engine.QueryRow(txsql+" WHERE tx_key = ?", key)
	var hash string
	var val []byte
	var ver int64
	var st int64
	err := row.Scan(&hash, &val, &ver, &st)
	if err == sql.ErrNoRows {
		return "", nil, -1, nil
	}

	if err != nil {
		return "", nil, -1, err
	}

	return hash, val, st, nil
}

func (db *MySqlDB) TxGetByAddr(addr string, index int, size int) ([]string, [][]byte, []int64, error) {
	start := (index - 1) * size
	stmt, err := db.engine.Prepare(txsql + " WHERE tx_from = ? or tx_to = ? ORDER BY tx_ver ASC LIMIT ?, ?")
	if err != nil {
		return nil, nil, nil, err
	}

	rows, err := stmt.Query(addr, addr, start, size)
	if err != nil {
		return nil, nil, nil, err
	}

	resultHash := make([]string, 0)
	resultVal := make([][]byte, 0)
	resultSt := make([]int64, 0)
	for rows.Next() {
		var hash string
		var val []byte
		var ver int64
		var st int64
		err := rows.Scan(&hash, &val, &ver, &st)
		if err != nil {
			return nil, nil, nil, err
		} else {
			resultHash = append(resultHash, hash)
			resultVal = append(resultVal, val)
			resultSt = append(resultSt, st)
		}
	}
	return resultHash, resultVal, resultSt, nil
}

func (db *MySqlDB) Clear() error {
	_, err := db.engine.Exec("DELETE FROM t_kv")
	if err != nil {
		return err
	}

	_, err = db.engine.Exec("DELETE FROM t_tx")
	if err != nil {
		return err
	}

	return nil
}

func (db *MySqlDB) Close() {
	db.engine.Close()
}
