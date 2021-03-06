// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/binding"
	"github.com/macaron-contrib/session"

	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/modules/log"
	"github.com/gogits/gogs/modules/setting"
)

// SignedInId returns the id of signed in user.
func SignedInId(header http.Header, sess session.Store) int64 {
	if !models.HasEngine {
		return 0
	}

	if setting.Service.EnableReverseProxyAuth {
		webAuthUser := header.Get(setting.ReverseProxyAuthUser)
		if len(webAuthUser) > 0 {
			u, err := models.GetUserByName(webAuthUser)
			if err != nil {
				if err != models.ErrUserNotExist {
					log.Error(4, "GetUserByName: %v", err)
				}
				return 0
			}
			return u.Id
		}
	}

	uid := sess.Get("uid")
	if uid == nil {
		return 0
	}
	if id, ok := uid.(int64); ok {
		if _, err := models.GetUserById(id); err != nil {
			if err != models.ErrUserNotExist {
				log.Error(4, "GetUserById: %v", err)
			}
			return 0
		}
		return id
	}
	return 0
}

// SignedInUser returns the user object of signed user.
func SignedInUser(header http.Header, sess session.Store) *models.User {
	uid := SignedInId(header, sess)
	if uid <= 0 {
		return nil
	}

	u, err := models.GetUserById(uid)
	if err != nil {
		log.Error(4, "GetUserById: %v", err)
		return nil
	}
	return u
}

type Form interface {
	binding.Validator
}

// AssignForm assign form values back to the template data.
func AssignForm(form interface{}, data map[string]interface{}) {
	typ := reflect.TypeOf(form)
	val := reflect.ValueOf(form)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		fieldName := field.Tag.Get("form")
		// Allow ignored fields in the struct
		if fieldName == "-" {
			continue
		}

		data[fieldName] = val.Field(i).Interface()
	}
}

func getSize(field reflect.StructField, prefix string) string {
	for _, rule := range strings.Split(field.Tag.Get("binding"), ";") {
		if strings.HasPrefix(rule, prefix) {
			return rule[8 : len(rule)-1]
		}
	}
	return ""
}

func GetMinSize(field reflect.StructField) string {
	return getSize(field, "MinSize(")
}

func GetMaxSize(field reflect.StructField) string {
	return getSize(field, "MaxSize(")
}

func validate(errs binding.Errors, data map[string]interface{}, f Form, l macaron.Locale) binding.Errors {
	if errs.Len() == 0 {
		return errs
	}

	data["HasError"] = true
	AssignForm(f, data)

	typ := reflect.TypeOf(f)
	val := reflect.ValueOf(f)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		fieldName := field.Tag.Get("form")
		// Allow ignored fields in the struct
		if fieldName == "-" {
			continue
		}

		if errs[0].FieldNames[0] == field.Name {
			data["Err_"+field.Name] = true
			trName := l.Tr("form." + field.Name)
			switch errs[0].Classification {
			case binding.RequiredError:
				data["ErrorMsg"] = trName + l.Tr("form.require_error")
			case binding.AlphaDashError:
				data["ErrorMsg"] = trName + l.Tr("form.alpha_dash_error")
			case binding.AlphaDashDotError:
				data["ErrorMsg"] = trName + l.Tr("form.alpha_dash_dot_error")
			case binding.MinSizeError:
				data["ErrorMsg"] = trName + l.Tr("form.min_size_error", GetMinSize(field))
			case binding.MaxSizeError:
				data["ErrorMsg"] = trName + l.Tr("form.max_size_error", GetMaxSize(field))
			case binding.EmailError:
				data["ErrorMsg"] = trName + l.Tr("form.email_error")
			case binding.UrlError:
				data["ErrorMsg"] = trName + l.Tr("form.url_error")
			default:
				data["ErrorMsg"] = l.Tr("form.unknown_error") + " " + errs[0].Classification
			}
			return errs
		}
	}
	return errs
}
