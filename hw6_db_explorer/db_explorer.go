package main

import (
	"net/http"
	"net/url"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"strconv"
)

type Handler struct {
	DB 			*sql.DB
	tables 	map[string]Table
}

var (
	entriesPattern = regexp.MustCompile(`^\/[[:alnum:]]+$`)
	entryPattern = regexp.MustCompile(`^\/[[:alnum:]]+\/.+`)
)

type ApiErrorResponse struct {
	Error string `json:"error"`
}

type ApiSuccessResponse struct {
	Error 		string 			`json:"error"`
	Response 	interface{} `json:"response"`
}

type FieldInfo struct {
	Name				string
	Type    		string
	Nullable 		string	
}

type Table struct {
	PrimaryKey		string
	Fields				map[string]Field
}

type Field struct {
	Name				string
	Type				string
	Nullable		bool
}

func InternalServerError(err error, w http.ResponseWriter) {
	fmt.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func errorResponse(status int, message string, w http.ResponseWriter) {
	res, err := json.Marshal(ApiErrorResponse{message})
	if err != nil {
		InternalServerError(err, w)
		return
	}
	w.WriteHeader(status)	
	w.Write(res)
}

func successResponse(status int, obj interface{}, w http.ResponseWriter) {
	res, err := json.Marshal(ApiSuccessResponse{"", obj})
	if err != nil {
		InternalServerError(err, w)
		return
	}
	w.WriteHeader(status)	
	w.Write(res)	
}

func (h *Handler) ListTables(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SHOW TABLES;")
	defer rows.Close()
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	
	tables := make([]string, 0)
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			InternalServerError(err, w)			
			return
		}
		tables = append(tables, table)
	}	

	successResponse(http.StatusOK, map[string][]string{"tables": tables}, w)
}

func getLimitOffset(u *url.URL) (string, string) {
	q := u.Query()
	limit, ok := q["limit"]
	if !ok {
		limit = []string{"5"}
	}
	offset, ok := q["offset"]
	if !ok {
		offset = []string{"0"}
	}

	return limit[0], offset[0]
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tableName := strings.Split(r.URL.Path, "/")[1]	
	table, ok := h.tables[tableName]
	if !ok {
		errorResponse(http.StatusBadRequest, "Invalid table param", w)
		return
	}
	limit, offset := getLimitOffset(r.URL)	

	rows, err := h.DB.Query("SELECT * FROM " + tableName + " LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	defer rows.Close()

	result, err := parseResponse(rows, &table)
	if err != nil {
		InternalServerError(err, w)			
		return
	}		

	successResponse(http.StatusOK, map[string]interface{}{"records": result}, w)
}

func parseResponse(rows *sql.Rows, table *Table) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	count := len(columns)
	
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)
		for i := range columns {			
			valuePtrs[i] = &values[i]
		}

		err = rows.Scan(valuePtrs...)		
		if err != nil {		
			return nil, err
		}		
		result = append(result, parseRow(table, columns, values))
	}	

	return result, nil
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(params[2])
	if err != nil {
		errorResponse(http.StatusBadRequest, "id must be int", w)
		return
	}

	tableName := params[1]
	table, ok := h.tables[tableName]
	if !ok {
		errorResponse(http.StatusBadRequest, "Invalid table param", w)
		return
	}
	rows, err := h.DB.Query("SELECT * FROM " + tableName + " WHERE " + h.tables[tableName].PrimaryKey + " = ? LIMIT 1", id)
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	defer rows.Close()

	result, err := parseResponse(rows, &table)
	if err != nil {
		InternalServerError(err, w)			
		return
	}	
	if len(result) == 0 {
		errorResponse(http.StatusNotFound, "Not Found", w)
		return 
	}

	successResponse(http.StatusOK, map[string]interface{}{"record": result[0]}, w)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	
}

func strToBool(in string) bool {
	if in == "YES" {
		return true
	}
	return false
}

type NullString struct {
	Valid		bool
	String 	string
}

func (s NullString) MarshalJSON() ([]byte, error) {
	if s.Valid {
		return []byte("\"" + s.String + "\""), nil
	}	
	return []byte("null"), nil
}

func parseRow(table *Table, columns []string, vals []interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for i, val := range vals {
		fieldName := columns[i]
		field, ok := table.Fields[fieldName]
		if !ok {
			panic("Can't find field description")
		}

		switch {
		case field.Type == "string":			
			b, ok := val.([]byte)
			if ok {
				result[fieldName] = NullString{true, string(b)}
			} else {
				result[fieldName] = NullString{false, ""}
			}			
		case field.Type == "int":			
			result[fieldName] = val							
		}
	}

	return result
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(params[2])
	if err != nil {
		errorResponse(http.StatusBadRequest, "id must be int", w)
		return
	}
	res, err := h.DB.Exec("DELETE FROM " + params[1] + " WHERE id = ?;", id)	
	if err != nil {
		fmt.Println(err)
		InternalServerError(err, w)		
		return
	}
	
	affected, err := res.RowsAffected() 
	if err != nil {
		InternalServerError(err, w)		
		return
	}	

	successResponse(http.StatusOK, map[string]int64{"deleted": affected}, w)
}

func (h *Handler) LoadMetadata() {
	h.tables = map[string]Table{}
	rows, err := h.DB.Query("SHOW TABLES;")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	
	tables := make([]string, 0)
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			panic(err)
		}
		tables = append(tables, table)
	}	

	for _, tableName := range tables {
		table := Table{}
		
		rows, err := h.DB.Query("SHOW FULL COLUMNS FROM " + tableName + ";")
		if err != nil {
			panic(err)
		}

		var skipColumn	sql.NullString
		var key 				sql.NullString
		fields := map[string]Field{}
		for rows.Next() {
			var fieldInfo FieldInfo
			err = rows.Scan(
				&fieldInfo.Name, 
				&fieldInfo.Type, 
				&skipColumn,
				&fieldInfo.Nullable,
				&key,
				&skipColumn,
				&skipColumn,
				&skipColumn,
				&skipColumn,
			)
			if err != nil {
				panic(err)
			}
			if key.Valid && key.String == "PRI" {
				table.PrimaryKey = fieldInfo.Name
			}

			fields[fieldInfo.Name] = Field{
				Name: 		fieldInfo.Name,
				Type: 		sqlTypeToGolangType(fieldInfo.Type),
				Nullable: strToBool(fieldInfo.Nullable),
			}			
		}
		table.Fields = fields
		h.tables[tableName] = table		
	}
}

func sqlTypeToGolangType(sqlType string) string {
	switch {
		case sqlType == "int":
			return "int"
		case sqlType == "text" || strings.Contains(sqlType, "varchar"):
			return "string"
		default:
			panic("Undefined type")
	}				
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request: ", r.Method, r.URL.Path)

	switch {
	case r.Method == "GET" && r.URL.Path == "/":
		h.ListTables(w, r)
	case r.Method == "GET" && entriesPattern.MatchString(r.URL.Path):
		h.List(w, r)
	case r.Method == "GET" && entryPattern.MatchString(r.URL.Path):
		h.Show(w, r)
	case r.Method == "PUT" && entriesPattern.MatchString(r.URL.Path):
		h.Create(w, r)
	case r.Method == "POST" && entryPattern.MatchString(r.URL.Path):
		h.Edit(w, r)
	case r.Method == "DELETE" && entryPattern.MatchString(r.URL.Path):
		h.Delete(w, r)					
	default:
		errorResponse(http.StatusNotFound, "unknown method", w)
	}
}

func NewDbExplorer(db *sql.DB) (http.Handler, error)  {	
	handler := Handler{DB: db}
	handler.LoadMetadata()

	return handler, nil
}
