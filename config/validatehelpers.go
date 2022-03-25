/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
)

//https://ahmet.im/blog/golang-take-slices-of-any-type-as-input-parameter/
func toSliceInterface(s interface{}) (out []interface{}, ok bool) {
	slice := reflect.ValueOf(s)
	if slice.Kind() != reflect.Slice {
		ok = false
		return
	}
	c := slice.Len()
	out = make([]interface{}, c)
	for i := 0; i < c; i++ {
		out[i] = slice.Index(i).Interface()
	}
	return out, true
}

func validateURL(u string) error {
	if u == "" {
		return errors.New("empty input url not allowed")
	}
	check, err := url.Parse(u)
	if err != nil {
		return err
	}
	host := check.Hostname()
	if host == "" || host == "0.0.0.0" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", u, err)
	}
	if addr := net.ParseIP(check.Hostname()); addr == nil {
		if _, err := net.LookupHost(check.Hostname()); err != nil {
			return err
		}
	}
	return nil
}

func checkDuplicates(val interface{}) error {
	slice, ok := toSliceInterface(val)
	if !ok {
		return errors.New("couldnt convert to slice interface{}")
	}
	check := make(map[string]int)
	for _, i := range slice {
		var key, name string
		var u *url.URL
		var err error
		switch v := i.(type) {
		case Input:
			key = v.Url
			name = "inputs"
			u, err = url.Parse(key)
			if err != nil {
				return err
			}
		case Output:
			key = v.Url
			name = "outputs"
			u, err = url.Parse(key)
			if err != nil {
				return err
			}
		case Flow:
			key = v.Identifier
		}
		if _, ok := check[key]; ok {
			return fmt.Errorf("duplicate url: %s in %s", key, name)
		}
		if u != nil {
			if _, ok := check[u.Host]; ok {
				return fmt.Errorf("duplicate url: %s in %s", u.Host, name)
			}
			check[u.Host] = 1
		}
		check[key] = 1
	}
	return nil
}
