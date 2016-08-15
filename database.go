package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/mattn/go-oci8"
)

type databaseType int

const (
	cps databaseType = iota
	cemr
)

type databaseSettings struct {
	Type     databaseType
	Name     string
	Host     string
	Port     int
	Username string
	Password string
}

type patient struct {
	PID           int64
	PatientID     string
	FirstName     string
	LastName      string
	DateOfBirth   time.Time
	MaxDocumentID int64
}

type database struct {
	typ        databaseType
	connection *sql.DB
}

func newDatabase(settings databaseSettings) (database, error) {
	var connectionString string
	var driver string

	switch settings.Type {
	case cps:
		driver = "mssql"
		connectionString = fmt.Sprintf("server=%s;database=%s;", settings.Host, settings.Name)

		if settings.Username != "" {
			connectionString = connectionString + fmt.Sprintf("user id=%s;password=%s;", settings.Username, settings.Password)
		}

		if settings.Port != 1433 {
			connectionString = connectionString + fmt.Sprintf("port=%d;", settings.Port)
		}
	case cemr:
		driver = "oci8"
		connectionString = fmt.Sprintf("%s/%s@%s:%d/%s", settings.Username, settings.Password, settings.Host, settings.Port, settings.Name)
	}

	connection, err := sql.Open(driver, connectionString)

	if err != nil {
		return database{}, err
	}

	return database{typ: settings.Type, connection: connection}, nil
}

func (d *database) getPatientData(pids []int64) ([]patient, error) {
	var tableName string

	if len(pids) > 0 {
		tn, err := d.createPidTempTable(pids)

		if err != nil {
			return nil, err
		}

		defer func() { _ = d.dropPidTempTable(tn) }()
		tableName = tn
	} else {
		tableName = "document"
	}

	q := fmt.Sprintf(`
select
	pid as "pid",
	patientId as "patientid",
	firstname as "firstname",
	lastname as "lastname",
	dateofbirth as "dateofbirth",
	(select max(sdid) from document where pid = p.pid) as "sdid"
from person p
where pid in (select pid from %s)`, tableName)

	rows, err := d.connection.Query(q)

	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	patients := []patient{}

	for rows.Next() {
		var patient patient

		if err := rows.Scan(&patient.PID, &patient.PatientID, &patient.FirstName, &patient.LastName, &patient.DateOfBirth, &patient.MaxDocumentID); err != nil {
			return nil, err
		}

		patients = append(patients, patient)
	}

	return patients, nil
}

func (d *database) createPidTempTable(pids []int64) (string, error) {
	baseTableName := "thicketPidList"
	var q string
	var tableName string

	switch d.typ {
	case cps:
		q = fmt.Sprintf("create table #%s(pid numeric(19, 0))", baseTableName)
		tableName = "#" + baseTableName
	case cemr:
		q = fmt.Sprintf("create global temporary table %s(pid decimal)", baseTableName)
		tableName = baseTableName
	}

	_, err := d.connection.Exec(q)

	if err != nil {
		return "", err
	}

	for _, pid := range pids {
		_, err := d.connection.Exec(fmt.Sprintf("insert into %s values ($1)", tableName), pid)

		if err != nil {
			return "", err
		}
	}

	return "#pidList", nil
}

func (d *database) dropPidTempTable(name string) error {
	_, err := d.connection.Exec("drop table " + name)
	return err
}

func (d *database) getProviderID(loginname string) (int64, error) {
	var pvid int64
	err := d.connection.QueryRow("select pvid from usr where loginname=$1", loginname).Scan(&pvid)

	switch {
	case err == sql.ErrNoRows:
		return 0, fmt.Errorf("no provider found for loginname %s", loginname)
	case err != nil:
		return 0, err
	default:
		return pvid, nil
	}
}

func (d *database) createOrder(pid, sdid, pvid int64) (int64, error) {
	oid, err := d.getOrderID()

	if err != nil {
		return 0, err
	}

	_, err = d.connection.Exec(`
insert into orders (orderid, pid, sdid, lupd, authbyusrid, locofservice, istoc, usrid, pubuser)
values ($1, $2, $3, $3, $4, $4, 'Y', $4, $4)`, oid, pid, sdid, pvid)

	if err != nil {
		return 0, err
	}

	return oid, nil
}

func (d *database) getOrderID() (int64, error) {
	var q string

	switch d.typ {
	case cps:
		q = "exec GEN_EMR_ID"
	case cemr:
		q = "select GEN_EMR_ID from dual"
	}

	var oid int64

	err := d.connection.QueryRow(q).Scan(&oid)

	switch {
	case err == sql.ErrNoRows:
		return 0, fmt.Errorf("order id couldn't be generated")
	case err != nil:
		return 0, err
	default:
		return oid, nil
	}
}

func (d *database) deleteOrder(oid int64) error {
	_, err := d.connection.Exec("delete from orders where orderId = $1", oid)
	return err
}
