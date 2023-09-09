// File: "utmp_test.go"

package utmp

import (
	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUTMP(t *testing.T) {
	_, err := GetUsers("", true)
	require.NoError(t, err, "Can't get users from utmp file")

	l, err := NewLogin("", true)
	require.NoError(t, err, "Can't create 'Login' object")
	l.Close()
}

// EOF: "utmp_test.go"
