package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"bufio"

	jlexer "github.com/mailru/easyjson/jlexer"	
)

var Empty struct{}

//easyjson:json
type User struct{
	Name string `json:"name"`
	Email string `json:"email"`
	Browsers []string `json:"browsers"`
}

func easyjson3486653aDecodeUsers(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "name":
			out.Name = string(in.String())
		case "email":
			out.Email = string(in.String())
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *User) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson3486653aDecodeUsers(&r, v)
	return r.Error()
}

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	seenBrowsers := make(map[string]struct{})
	uniqueBrowsers := 0
	var foundUsers strings.Builder
	user := new(User)

	scanner := bufio.NewScanner(file)
	i := -1

	for scanner.Scan() {
		i++
		err := user.UnmarshalJSON([]byte(scanner.Text()))
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false
		for _, browser := range user.Browsers {
			switch {
			case strings.Contains(browser, "Android"):
				isAndroid = true
			case strings.Contains(browser, "MSIE"):
				isMSIE = true
			default:
				continue		
			}

			_, ok := seenBrowsers[browser]				
				if !ok {
					seenBrowsers[browser] = Empty
					uniqueBrowsers++
				}
		}

		if !(isAndroid && isMSIE) {
			continue
		}		
		
		foundUsers.WriteString(
			fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, strings.Replace(user.Email, "@", " [at] ", 1)),
		)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}