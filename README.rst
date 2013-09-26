testbed
=======

Runs test codes with service stubs for Google App Engine Go.

To test applications which use App Engine services such as the
datastore, developers can use the available stub
implementations. Service stubs behave like the original service
without permanent side effects. The datastore stub for example allows
to write entities into memory without storing them to the actual
datastore. This module makes using those stubs for testing easier.

Google App Engine SDK 1.7.5 or higher required.

Example
=======

::

  package yourapp
  
  import (
  	"appengine/datastore"
  	"net/http"
  	"testing"
  	"github.com/najeira/testbed"
  )
  
  const (
  	PYTHON    = `/usr/local/bin/python27`
  	TESTBED   = `/usr/local/google_appengine/goroot/src/pkg/github.com/najeira/testbed/testbed.py`
  	APPENGINE = `/usr/local/google_appengine`
  )
  
  func TestAllocateIDs(t *testing.T) {
  	bed := testbed.NewTestbed(PYTHON, TESTBED, APPENGINE)
  	bed.Start()
  	defer bed.Close()
  	
  	// create a dummy context
  	r, _ := http.NewRequest("GET", "http://example.com/", nil)
  	c := bed.NewContext(r)
  	
  	// write your test codes here
  	low, high, err := datastore.AllocateIDs(c, "Test", nil, 10)
  	if err != nil {
  		t.Errorf("got error: %v", err)
  	}
  	if high - low != 10 {
  		t.Errorf("wrong values: %d, %d", low, high)
  	}
  }


Note
====

To use testbed with 1.7.6 - 1.8.4, apply a patch to `appengine_internal/api_dev.go`.

::

  56,60c56,68
  < 	instanceConfig.AppID = string(c.AppId)
  < 	instanceConfig.APIPort = int(*c.ApiPort)
  < 	instanceConfig.VersionID = string(c.VersionId)
  < 	instanceConfig.InstanceID = *c.InstanceId
  < 	instanceConfig.Datacenter = *c.Datacenter
  ---
  > 	if c != nil {
  > 		instanceConfig.AppID = string(c.AppId)
  > 		instanceConfig.APIPort = int(*c.ApiPort)
  > 		instanceConfig.VersionID = string(c.VersionId)
  > 		instanceConfig.InstanceID = *c.InstanceId
  > 		instanceConfig.Datacenter = *c.Datacenter
  > 	} else {
  > 		instanceConfig.AppID = "app"
  > 		instanceConfig.APIPort = 54321
  > 		instanceConfig.VersionID = "version"
  > 		instanceConfig.InstanceID = "abc3dzac4"
  > 		instanceConfig.Datacenter = "us1"
  > 	}
  121a130,132
  > 	if len(raw) == 0 {
  > 		return nil
  > 	}
  247c258,259
  < func init() { os.DisableWritesForAppEngine = true }
  \ No newline at end of file
  ---
  >
  > // func init() { os.DisableWritesForAppEngine = true }  


License
=======

New BSD License.
