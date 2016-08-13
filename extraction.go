package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

func runExtraction(db database, a api, pids []int64, outputPath string) {
	stderr := log.New(os.Stderr, "", 0)
	stdout := log.New(os.Stdout, "", 0)

	pvid, err := db.getProviderID(a.Username)

	if err != nil {
		stderr.Fatal(err)
	}

	patientData, err := db.getPatientData(pids)

	if err != nil {
		stderr.Fatal(err)
	}

	for _, patient := range patientData {
		ccda, err := getPatientCcda(db, a, pvid, patient)

		if err != nil {
			stderr.Printf("Patient %d error %s\n", patient.PID, err)
			continue
		}

		p, err := writeFile(ccda, patient, outputPath)

		if err != nil {
			stderr.Printf("Patient %d error %s\n", patient.PID, err)
			continue
		}

		stdout.Printf("Patient %d written to %s\n", patient.PID, p)
	}
}

func getPatientCcda(db database, a api, pvid int64, patient patient) (ccda string, err error) {
	oid, err := db.createOrder(patient.PID, patient.MaxDocumentID, pvid)

	if err != nil {
		return
	}

	defer rollbackOrder(db, oid, &err)

	ccda, err = a.generateCcda(oid, patient.PID, patient.MaxDocumentID, pvid)

	if err != nil {
		return
	}

	return
}

func rollbackOrder(db database, oid int64, err *error) {
	derr := db.deleteOrder(oid)

	if derr != nil {
		*err = derr
	}
}

func writeFile(ccda string, patient patient, outputPath string) (string, error) {
	filename := strings.Join([]string{
		"CCDA" + patient.PatientID,
		patient.LastName,
		patient.FirstName,
		patient.DateOfBirth.Format("20060102150405") + ".xml",
	}, "_")

	filepath := path.Join(outputPath, filename)

	err := ioutil.WriteFile(filepath, []byte(ccda), os.ModePerm)

	if err != nil {
		return "", err
	}

	return filepath, nil
}
