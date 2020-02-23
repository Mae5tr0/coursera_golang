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
	"io/ioutil"
	"sort"
)

type Handler struct {
	DB 			*sql.DB
	tables 	map[string]Table
}

var (
	entriesPattern = regexp.MustCompile(`^\/[^\/]+\/?$`)
	entryPattern = regexp.MustCompile(`^\/[^\/]+\/.+`)
)

type ApiErrorResponse struct {
	Error string `json:"error"`
}

type ApiSuccessResponse struct {
	Response 	interface{} `json:"response"`
}

type Table struct {
	Name					string
	PrimaryKey		string
	Fields				map[string]Field
}

type Field struct {
	Name					string
	Type					string
	Nullable			bool
	AutoIncrement bool
	Default 			interface{}
}

const (
	DEFAULT_LIMIT = 5
	DEFAULT_OFFSET = 0
)

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
	res, err := json.Marshal(ApiSuccessResponse{obj})
	if err != nil {
		InternalServerError(err, w)
		return
	}
	w.WriteHeader(status)	
	w.Write(res)	
}

func (h *Handler) ListTables(w http.ResponseWriter, r *http.Request) {
	var tables []string
	for k := range h.tables {
			tables = append(tables, k)
	}
	sort.Strings(tables)

	successResponse(http.StatusOK, map[string][]string{"tables": tables}, w)
}

func getLimitOffset(u *url.URL) (int, int) {
	q := u.Query()
	limit := DEFAULT_LIMIT
	offset := DEFAULT_OFFSET

	limitParam, _ := q["limit"]
	if len(limitParam) > 0 {				
		limitInt, err := strconv.Atoi(limitParam[0])
		if err == nil {
			limit = limitInt
		}	
	}

	offsetParam, _ := q["offset"]
	if len(offsetParam) > 0 {				
		offsetInt, err := strconv.Atoi(offsetParam[0])
		if err == nil {
			offset = offsetInt
		}	
	}

	return limit, offset
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	table, err := getTable(r, h)
	if err != nil {
		errorResponse(http.StatusNotFound, err.Error(), w)
		return
	}
	limit, offset := getLimitOffset(r.URL)	

	rows, err := h.DB.Query("SELECT * FROM " + table.Name + " LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	defer rows.Close()

	result, err := parseResponse(rows, table)
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
	table, err := getTable(r, h)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}
	key, err := getKey(r)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}	

	rows, err := h.DB.Query("SELECT * FROM " + table.Name + " WHERE " + table.PrimaryKey + " = ? LIMIT 1", key)
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	defer rows.Close()

	result, err := parseResponse(rows, table)
	if err != nil {
		InternalServerError(err, w)			
		return
	}	
	if len(result) == 0 {
		errorResponse(http.StatusNotFound, "record not found", w)
		return 
	}

	successResponse(http.StatusOK, map[string]interface{}{"record": result[0]}, w)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {	
	table, err := getTable(r, h)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}
	
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	var params map[string]interface{}
	err = json.Unmarshal(body, &params)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	var fields []string
	var values  []interface{}

	for _,field := range table.Fields {
		if field.AutoIncrement {
			continue
		}		
		
		fields = append(fields, field.Name)
		value, _ := params[field.Name]
		if value == nil {
			values = append(values, field.Default)
			continue
		}
		values = append(values, value)			
	}

	insertSQL := "INSERT INTO $table_name (`$fields`) VALUES ($values)"
	insertSQL = strings.Replace(insertSQL, "$table_name", table.Name, 1)
	insertSQL = strings.Replace(insertSQL, "$fields", strings.Join(fields, "`, `"), 1)
	insertSQL = strings.Replace(insertSQL, "$values", strings.Repeat("?, ", len(fields) - 1) + "?", 1)
	fmt.Println(insertSQL)
	_, err = h.DB.Exec(insertSQL, values...)	
	if err != nil {
		fmt.Println(err)
		InternalServerError(err, w)		
		return
	}
	rows, err := h.DB.Query("SELECT LAST_INSERT_ID()")
	if err != nil {
		InternalServerError(err, w)		
		return
	}
	defer rows.Close()

	var key int
	for rows.Next() {		
		err = rows.Scan(&key)		
		if err != nil {	
			InternalServerError(err, w)		
			return				
		}				
	}

	successResponse(http.StatusOK, map[string]int{table.PrimaryKey: key}, w)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	table, err := getTable(r, h)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}
	key, err := getKey(r)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}	
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}
	var params map[string]interface{}
	err = json.Unmarshal(body, &params)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	var fields []string
	var values  []interface{}
	fmt.Printf("Params: %v",params)
	for key, value := range params {
		field, ok := table.Fields[key]
		if !ok { continue } // skip unknown fields
		fields = append(fields, field.Name)
		invalidTypeMessage := "field " + field.Name + " have invalid type"
		// update auto increment field does not allowed
		if field.AutoIncrement {
			errorResponse(http.StatusBadRequest, invalidTypeMessage, w)
			return
		}

		switch {
		case value == nil && field.Nullable:
			values = append(values, nil)				
		case field.Type == "string":
			val, ok := value.(string)
			if !ok {				
				errorResponse(http.StatusBadRequest, invalidTypeMessage, w)
				return
			}
			values = append(values, val)
		case field.Type == "int":
			val, ok := value.(int)
			if !ok {
				errorResponse(http.StatusBadRequest, invalidTypeMessage, w)
				return
			}
			values = append(values, val)
		}		
	}	
	values = append(values, key)

	fmt.Printf("\nFields: %v",fields)
	fmt.Printf("\nValues: %v",values)

	updateSQL := "UPDATE $table_name SET $fields = ? WHERE $primary_key = ?"
	updateSQL = strings.Replace(updateSQL, "$table_name", table.Name, 1)
	updateSQL = strings.Replace(updateSQL, "$primary_key", table.PrimaryKey, 1)
	updateSQL = strings.Replace(updateSQL, "$fields", strings.Join(fields, " = ?, "), 1)
	fmt.Println()
	fmt.Printf(updateSQL, values...)
	res, err := h.DB.Exec(updateSQL, values...)	
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

	successResponse(http.StatusOK, map[string]int64{"updated": affected}, w)
}

func getKey(r *http.Request) (int, error) {
	params := strings.Split(r.URL.Path, "/")
	key, err := strconv.Atoi(params[2])
	if err != nil {
		return 0, fmt.Errorf("Primary key must be int")
	}

	return key, nil
}

func getTable(r *http.Request, h *Handler) (*Table, error) {
	params := strings.Split(r.URL.Path, "/")	
	tableName := params[1]
	table, ok := h.tables[tableName]
	if !ok {
		return nil, fmt.Errorf("unknown table")
	}

	return &table, nil
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
		return json.Marshal(s.String)
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
	table, err := getTable(r, h)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}
	id, err := getKey(r)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}	
	
	res, err := h.DB.Exec("DELETE FROM " + table.Name + " WHERE id = ?;", id)	
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
		table := Table{
			Name: tableName,
		}
		
		rows, err := h.DB.Query("SHOW FULL COLUMNS FROM " + tableName + ";")
		if err != nil {
			panic(err)
		}

		var skipColumn	sql.NullString
		fields := map[string]Field{}
		for rows.Next() {
			var fieldInfo FieldInfo
			err = rows.Scan(
				&fieldInfo.Name, 
				&fieldInfo.Type, 
				&skipColumn,
				&fieldInfo.Nullable,
				&fieldInfo.Key,
				&fieldInfo.Default,
				&fieldInfo.Extra,
				&skipColumn,
				&skipColumn,
			)
			if err != nil {
				panic(err)
			}
			if fieldInfo.Key.Valid && fieldInfo.Key.String == "PRI" {
				table.PrimaryKey = fieldInfo.Name
			}
			
			field := Field{
				Name: 		fieldInfo.Name,
				Type: 		sqlTypeToGolangType(fieldInfo.Type),
				Nullable: strToBool(fieldInfo.Nullable),
				AutoIncrement: fieldInfo.Extra.Valid && fieldInfo.Extra.String == "auto_increment",
			}				
			if (fieldInfo.Default.Valid) {
				field.Default = fieldInfo.Default.String
			}			
			if field.Default == nil && !field.Nullable {
				switch field.Type {
				case "string":
					field.Default = ""
				case "int":
					field.Default = 0
				}	
			}
			fields[fieldInfo.Name] = field			
		}
		table.Fields = fields
		h.tables[tableName] = table		
	}
}

type FieldInfo struct {
	Name				string
	Type    		string
	Nullable 		string
	Key 				sql.NullString
	Extra 			sql.NullString
	Default				sql.NullString
}

func sqlTypeToGolangType(sqlType string) string {
	switch {
		case strings.Contains(sqlType, "int"):
			return "int"
		case sqlType == "text" || strings.Contains(sqlType, "varchar"):
			return "string"
		default:
			fmt.Println("Unknown type: ", sqlType)
			return "string"
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
