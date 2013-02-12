package testbed

import (
	"testing"
	"appengine/datastore"
	"net/http"
	"fmt"
)

const (
	PYTHON    = `C:\Python\27\python.exe`
	TESTBED   = `C:\Program Files (x86)\Google\google_appengine\goroot\src\pkg\github.com\najeira\testbed\testbed.py`
	APPENGINE = `C:\Program Files (x86)\Google\google_appengine`
)

func TestTestbed(t *testing.T) {
	bed := NewTestbed(PYTHON, TESTBED, APPENGINE)
	bed.Start()
	defer bed.Close()
	
	// create a dummy context
	r, _ := http.NewRequest("GET", "http://example.com/", nil)
	c := bed.NewContext(r)
	
	low, high, err := datastore.AllocateIDs(c, "Test", nil, 10)
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if high - low != 10 {
		t.Errorf("wrong values: %d, %d", low, high)
	}
	fmt.Printf("low=%d, high=%d\n", low, high)
	
	low2, high2, err := datastore.AllocateIDs(c, "Test", nil, 10)
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if high2 - low2 != 10 {
		t.Errorf("wrong values: %d, %d", low2, high2)
	}
	if low2 < high {
		t.Errorf("wrong values: %d, %d", high, low2)
	}
	fmt.Printf("low=%d, high=%d\n", low2, high2)
	
	bed.Reset()
	
	low3, high3, _ := datastore.AllocateIDs(c, "Test", nil, 10)
	fmt.Printf("low=%d, high=%d\n", low3, high3)
}
