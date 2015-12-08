package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFetchPage(t *testing.T) {
	data, err := fetchPage(0)
	assert.Nil(t, err)
	assert.NotNil(t, data)
}

func TestFetchData(t *testing.T) {
	things, _ := getData(0)
	assert.Len(t, things, 25)
}
