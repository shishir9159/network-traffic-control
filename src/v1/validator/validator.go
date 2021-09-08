package validator

import (
"fmt"
"log"
"reflect"
"regexp"
"strconv"
)

const (
	TRUE        = "true"
	FALSE       = "false"
	TYPE        = "type"
	EMAIL       = "email"
	STRING      = "string"
	NUMBER      = "number"
	STRUCT="struct"
	REQUIRED    = "required"
	MESSAGE     = "msg"
	REGEX       = "regex"
	EMAIL_REGEX = "^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"
	LENGTH      = "length"
	MAX_LENGTH  = "max_len"
	MIN_LENGTH  = "min_len"
	MAX_VALUE   = "max_val"
	MIN_VALUE   = "min_val"
)

type Validator struct {
	l *log.Logger
}

func NewValidatorWithLogger(l *log.Logger) *Validator {
	return &Validator{l}
}

func NewValidator() *Validator {
	return &Validator{}
}

func (v Validator) Struct(object interface{}) []string {
	if v.l != nil {
		log.Println(v.l.Prefix())
	}
	errs := []string{}
	obj := reflect.ValueOf(object)
	for i := 0; i < obj.Type().NumField(); i++ {
		required := obj.Type().Field(i).Tag.Get(REQUIRED)
		dataType := obj.Type().Field(i).Tag.Get(TYPE)
		value := fmt.Sprintf("%v", obj.Field(i))
		msg := obj.Type().Field(i).Tag.Get(MESSAGE)
		length := obj.Type().Field(i).Tag.Get(LENGTH)
		max_length := obj.Type().Field(i).Tag.Get(MAX_LENGTH)
		min_length := obj.Type().Field(i).Tag.Get(MIN_LENGTH)
		max_value := obj.Type().Field(i).Tag.Get(MAX_VALUE)
		min_value := obj.Type().Field(i).Tag.Get(MIN_VALUE)

		if required == "true" {
			if msg == "" {
				if dataType == EMAIL {
					if v.checkIfAgainstRegex(value, EMAIL_REGEX) == false {
						errs = append(errs, "Please provide a valid "+obj.Type().Field(i).Name+"! [ERROR]:Invalid email "+value+"!")
					}
				} else if dataType == REGEX {
					if v.checkIfAgainstRegex(value, EMAIL_REGEX) == false {
						errs = append(errs, "Please provide a valid "+obj.Type().Field(i).Name+"! [ERROR]:Invalid email "+value+"!")
					}
				} else if dataType==STRUCT{
					if value=="{ }"{
						errs = append(errs, "Struct: "+obj.Type().Field(i).Name+", cannot be empty! [ERROR]: Empty struct detected!")
					}

				} else {
					typeStr := reflect.TypeOf(obj.Type().Field(i).Name).String()
					if typeStr == STRING {
						if length != "" {
							i, err := strconv.Atoi(length)
							if err != nil {
								errs = append(errs, err.Error())
							} else {
								if len(value) != i {
									errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: Invalid size of "+value+"!")
								}
							}
						} else if max_length != "" && min_length != "" {
							max, max_err := strconv.Atoi(max_length)
							min, min_err := strconv.Atoi(min_length)

							if max_err != nil || min_err != nil {
								if max_err != nil {
									errs = append(errs, max_err.Error())
								}
								if min_err != nil {
									errs = append(errs, min_err.Error())
								}
							} else {
								if len(value) < min {
									errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_length+"!")
								}
								if len(value) > max {
									errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is larger than size : "+max_length+"!")
								}
							}
						} else if max_length != "" {
							max, max_err := strconv.Atoi(max_length)
							if max_err != nil {
								if max_err != nil {
									errs = append(errs, max_err.Error())
								}
							} else {
								if len(value) > max {
									errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is larger than size : "+max_length+"!")
								}
							}
						} else if min_length != "" {
							min, min_err := strconv.Atoi(min_length)

							if min_err != nil {
								if min_err != nil {
									errs = append(errs, min_err.Error())
								}
							} else {
								if len(value) < min {
									errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_length+"!")
								}
							}
						}else if  value==""{
							errs = append(errs, "Please provide a value! [ERROR]: "+obj.Type().Field(i).Name+" is empty!")
						}
					} else if dataType == NUMBER {
						if max_value != "" && min_value != "" {

							typeStr := reflect.TypeOf(obj.Type().Field(i).Name).String()
							if typeStr == "int" || typeStr == "uint" {
								max, max_err := strconv.Atoi(max_value)
								min, min_err := strconv.Atoi(min_value)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.Atoi(value)
									if val < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if val > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}
							} else if typeStr == "int8" || typeStr == "uint8" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int8(i)
									min := int8(i2)
									val, _ := strconv.Atoi(value)
									if int8(val) < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if int8(val) > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}
							} else if typeStr == "int16" || typeStr == "uint16" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int16(i)
									min := int16(i2)
									val, _ := strconv.Atoi(value)
									if int16(val) < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if int16(val) > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}
							} else if typeStr == "int32" || typeStr == "uint32" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int32(i)
									min := int32(i2)
									val, _ := strconv.Atoi(value)
									if int32(val) < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if int32(val) > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}
							} else if typeStr == "int64" || typeStr == "uint64" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int64(i)
									min := int64(i2)
									val, _ := strconv.Atoi(value)
									if int64(val) < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if int64(val) > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}
							} else if typeStr == "float32" {
								max, max_err := strconv.ParseFloat(max_value, 32)
								min, min_err := strconv.ParseFloat(min_value, 32)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.ParseFloat(value, 32)
									if val < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if val > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}

							} else if typeStr == "float64" {
								max, max_err := strconv.ParseFloat(max_value, 64)
								min, min_err := strconv.ParseFloat(min_value, 64)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.ParseFloat(value, 64)
									if val < min {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is smaller than size : "+min_value+"!")
									}
									if val > max {
										errs = append(errs, "Please provide a valid sized "+obj.Type().Field(i).Name+"! [ERROR]: "+value+" is greater than size : "+max_value+"!")
									}
								}

							}

						}

					}
					//else{
					//	errs = append(errs, "Please provide "+obj.Type().Field(i).Name+"![ERROR]: empty "+obj.Type().Field(i).Name+"!")
					//}

				}

			} else {
				if dataType == EMAIL {
					if v.checkIfAgainstRegex(value, EMAIL_REGEX) == false {
						errs = append(errs, "[ERROR]:"+msg)
					}
				} else if dataType == REGEX {
					if v.checkIfAgainstRegex(value, EMAIL_REGEX) == false {
						errs = append(errs, "[ERROR]:"+msg)
					}
				}else {
					typeStr := reflect.TypeOf(obj.Type().Field(i).Name).String()
					if typeStr == STRING {
						if length != "" {
							i, err := strconv.Atoi(length)
							if err != nil {
								errs = append(errs, err.Error())
							} else {
								if len(value) != i {
									errs = append(errs, msg)
								}
							}
						} else if max_length != "" && min_length != "" {
							max, max_err := strconv.Atoi(max_length)
							min, min_err := strconv.Atoi(min_length)

							if max_err != nil || min_err != nil {
								if max_err != nil {
									errs = append(errs, max_err.Error())
								}
								if min_err != nil {
									errs = append(errs, min_err.Error())
								}
							} else {
								if len(value) < min {
									errs = append(errs, msg)
								}
								if len(value) > max {
									errs = append(errs, msg)
								}
							}
						} else if max_length != "" {
							max, max_err := strconv.Atoi(max_length)
							if max_err != nil {
								if max_err != nil {
									errs = append(errs, max_err.Error())
								}
							} else {
								if len(value) > max {
									errs = append(errs, msg)
								}
							}
						} else if min_length != "" {
							min, min_err := strconv.Atoi(min_length)

							if min_err != nil {
								if min_err != nil {
									errs = append(errs, min_err.Error())
								}
							} else {
								if len(value) < min {
									errs = append(errs, msg)
								}
							}
						}else if  value==""{
							errs = append(errs, msg)
						}
					}	else if dataType == NUMBER {
						if max_value != "" && min_value != "" {

							typeStr := reflect.TypeOf(obj.Type().Field(i).Name).String()
							if typeStr == "int" || typeStr == "uint" {
								max, max_err := strconv.Atoi(max_value)
								min, min_err := strconv.Atoi(min_value)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.Atoi(value)
									if val < min {
										errs = append(errs, msg)
									}
									if val > max {
										errs = append(errs, msg)
									}
								}
							} else if typeStr == "int8" || typeStr == "uint8" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int8(i)
									min := int8(i2)
									val, _ := strconv.Atoi(value)
									if int8(val) < min {
										errs = append(errs, msg)
									}
									if int8(val) > max {
										errs = append(errs, msg)
									}
								}
							} else if typeStr == "int16" || typeStr == "uint16" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int16(i)
									min := int16(i2)
									val, _ := strconv.Atoi(value)
									if int16(val) < min {
										errs = append(errs, msg)
									}
									if int16(val) > max {
										errs = append(errs, msg)
									}
								}
							} else if typeStr == "int32" || typeStr == "uint32" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int32(i)
									min := int32(i2)
									val, _ := strconv.Atoi(value)
									if int32(val) < min {
										errs = append(errs, msg)
									}
									if int32(val) > max {
										errs = append(errs, msg)
									}
								}
							} else if typeStr == "int64" || typeStr == "uint64" {
								i, max_err := strconv.Atoi(max_value)
								i2, min_err := strconv.Atoi(min_value)
								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									max := int64(i)
									min := int64(i2)
									val, _ := strconv.Atoi(value)
									if int64(val) < min {
										errs = append(errs, msg)
									}
									if int64(val) > max {
										errs = append(errs, msg)
									}
								}
							} else if typeStr == "float32" {
								max, max_err := strconv.ParseFloat(max_value, 32)
								min, min_err := strconv.ParseFloat(min_value, 32)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.ParseFloat(value, 32)
									if val < min {
										errs = append(errs, msg)
									}
									if val > max {
										errs = append(errs, msg)
									}
								}

							} else if typeStr == "float64" {
								max, max_err := strconv.ParseFloat(max_value, 64)
								min, min_err := strconv.ParseFloat(min_value, 64)

								if max_err != nil || min_err != nil {
									if max_err != nil {
										errs = append(errs, max_err.Error())
									}
									if min_err != nil {
										errs = append(errs, min_err.Error())
									}
								} else {
									val, _ := strconv.ParseFloat(value, 64)
									if val < min {
										errs = append(errs, msg)
									}
									if val > max {
										errs = append(errs, msg)
									}
								}

							}

						}

					}
				}
			}

		}

	}

	return errs

}

func (v Validator) checkIfAgainstRegex(str string, regex string) bool {
	if len(str) < 3 && len(str) > 254 {
		return false
	}
	return regexp.MustCompile(regex).MatchString(str)
}
