package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"vms/api/dto"
)

func RespondWithSuccessString(writer http.ResponseWriter, value string, code int) error {
	writer.WriteHeader(code)

	responseMessage := []byte(value)

	_, err := writer.Write(responseMessage)
	if err != nil {
		return err
	}

	return nil
}

func RespondWithSuccessJson(writer http.ResponseWriter, value any, code int) error {
	responseMessage, err := json.MarshalIndent(&value, "", "    ")
	if err != nil {
		log.Panicln(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	_, err = writer.Write(responseMessage)
	if err != nil {
		return err
	}

	return nil
}

func RespondWithErrorString(writer http.ResponseWriter, message string, code int) {
	http.Error(writer, message, code)
}

func RespondWithErrorJson(writer http.ResponseWriter, message string, code int) {
	errDTO := dto.NewErrorDTO(message, time.Now())
	http.Error(writer, errDTO.ToString(), code)
}
