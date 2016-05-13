package pdft

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

const dictionary = "dictionary"
const object = "object"
const array = "array"
const number = "number"

//PDFObjPropertyData property of pdf obj
type PDFObjPropertyData struct {
	key    string
	rawVal string
}

func (p *PDFObjPropertyData) asDictionary() (int, int, error) {
	return readObjIDFromDictionary(p.rawVal)
}

func (p *PDFObjPropertyData) asDictionaryArr() ([]int, []int, error) {
	return readObjIDFromDictionaryArr(p.rawVal)
}

func (p *PDFObjPropertyData) valType() string {
	return propertyType(p.rawVal)
}

func propertyType(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > len("<<") && raw[0:len("<<")] == "<<" {
		return object
	} else if len(raw) > len("[") && raw[0:len("[")] == "[" {
		return array
	} else if _, err := strconv.Atoi(raw); err == nil {
		return number
	}
	//fmt.Printf("raw=%s\n", raw)
	return dictionary
}

func readProperty(rawObj *[]byte, key string) (*PDFObjPropertyData, error) {
	var outProps PDFObjPropertiesData
	err := readProperties(rawObj, &outProps)
	if err != nil {
		return nil, err
	}
	return outProps.getPropByKey(key), nil
}

func readProperties(rawObj *[]byte, outProps *PDFObjPropertiesData) error {

	startObjInx := strings.Index(string(*rawObj), "<<")
	endObjInx := strings.LastIndex(string(*rawObj), ">>")
	if startObjInx > endObjInx {
		return errors.New("bad obj properties")
	}

	var regexpSlash = regexp.MustCompile("[\\n\\t ]+\\/")
	var regexpOpenB = regexp.MustCompile("[\\n\\t ]+\\[")
	var regexpCloseB = regexp.MustCompile("[\\n\\t ]+\\]")
	var regexpOpen = regexp.MustCompile("[\\n\\t ]+\\<\\<")
	var regexpClose = regexp.MustCompile("[\\n\\t ]+\\>\\>")
	var regexpLine = regexp.MustCompile("[\\n\\t ]+")

	tmp := strings.TrimSpace(string((*rawObj)[startObjInx+len("<<") : endObjInx]))
	tmp = regexpLine.ReplaceAllString(tmp, " ")
	tmp = regexpSlash.ReplaceAllString(tmp, "/")
	tmp = regexpOpenB.ReplaceAllString(tmp, "[")
	tmp = regexpCloseB.ReplaceAllString(tmp, "]")
	tmp = regexpOpen.ReplaceAllString(tmp, "<<")
	tmp = regexpClose.ReplaceAllString(tmp, ">>")

	var pp parseProps
	pp.set(tmp, outProps)
	return nil
}

type parseProps struct {
	str        string
	max        int
	propsIndex int
	props      *PDFObjPropertiesData
}

func (p *parseProps) set(str string, props *PDFObjPropertiesData) {
	p.str = str
	p.max = len(str)
	p.propsIndex = -1
	p.props = props
	p.loop(0, "")
}

func (p *parseProps) loop(i int, status string) (int, string) {
	count01 := 0
	count02 := 0
	for i < p.max {
		r := string(p.str[i])
		if status == "" && r == "/" {
			p.propsIndex++
			p.props.append(PDFObjPropertyData{})
			i, status = p.loop(i+1, "key")
		} else if status == "key" {
			if r == " " {
				i, status = p.loop(i+1, "val")
			} else if r == "<" || r == "[" {
				i, status = p.loop(i, "val")
			} else if r == "/" {
				return i - 1, ""
			} else {
				p.props.at(p.propsIndex).key += r
			}
		} else if status == "val" {

			if r == "<" {
				count01++
			} else if r == "[" {
				count02++
			} else if r == ">" {
				count01--
			} else if r == "]" {
				count02--
			}

			if (r == "]" || r == ">") && (count01 == 0 && count02 == 0) {
				p.props.at(p.propsIndex).rawVal += r
				return i, ""
			} else if r == "/" && (count01 == 0 && count02 == 0) {
				return i - 1, ""
			}
			p.props.at(p.propsIndex).rawVal += r

		}
		i++
	}
	return i, status
}

func readObjIDFromDictionaryArr(str string) ([]int, []int, error) {

	str = strings.Replace(str, "[", "", -1)
	str = strings.Replace(str, "]", "", -1)
	str = strings.TrimSpace(str)
	tokens := strings.Split(str, " ")
	var objIDs []int
	var revisions []int

	i := 0
	max := len(tokens)
	for i < max {
		objID, err := strconv.Atoi(strings.TrimSpace(tokens[i]))
		if err != nil {
			return nil, nil, err
		}
		revision, err := strconv.Atoi(strings.TrimSpace(tokens[i+1]))
		if err != nil {
			return nil, nil, err
		}
		objIDs = append(objIDs, objID)
		revisions = append(revisions, revision)
		i += 3
	}

	return objIDs, revisions, nil
}

func readObjIDFromDictionary(str string) (int, int, error) {

	str = strings.TrimSpace(str)
	if str == "" {
		return 0, 0, errors.New("Object ID not found")
	}

	tokens := strings.Split(str, " ")
	if len(tokens) != 3 {
		return 0, 0, errors.New("Object ID not found")
	}

	id, err := strconv.Atoi(strings.TrimSpace(tokens[0]))
	if err != nil {
		return 0, 0, err
	}
	return id, 0, nil
}
