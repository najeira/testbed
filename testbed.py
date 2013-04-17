import sys
import logging
import base64
import pickle
import traceback

#logging.basicConfig(filename="./testbed.log", level=logging.DEBUG)

def setup_appengine():
  sdk_path = r"C:\Program Files (x86)\Google\google_appengine"
  if len(sys.argv) >= 2:
    sdk_path = sys.argv[1]
  sys.path.insert(0, sdk_path)
  from api_server import fix_sys_path, API_SERVER_EXTRA_PATHS
  fix_sys_path(API_SERVER_EXTRA_PATHS)
  if "google" in sys.modules:
    del sys.modules["google"]

setup_appengine()

from google.appengine.tools import api_server
from google.appengine.ext.remote_api import remote_api_pb
from google.appengine.ext import testbed
from google.appengine.runtime import apiproxy_errors
from google.appengine.datastore import datastore_stub_util

class Server(object):
  def setUp(self):
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    consistency_policy = datastore_stub_util.PseudoRandomHRConsistencyPolicy(1.0)
    self.testbed.init_datastore_v3_stub(consistency_policy=consistency_policy)
    self.testbed.init_memcache_stub()
    self.testbed.init_images_stub()
    self.testbed.init_mail_stub()
    self.testbed.init_taskqueue_stub()
    self.testbed.init_urlfetch_stub()
  
  def tearDown(self):
    self.testbed.deactivate()

def process(req):
  """Handles a single API request. from api_server.py"""
  response = remote_api_pb.Response()
  try:
    request = remote_api_pb.Request()
    request.ParseFromString(req)
    api_response = api_server._ExecuteRequest(request).Encode()
    response.set_response(api_response)
  except Exception, e:
    logging.error("Exception while handling %s\n%s",
                  request,
                  traceback.format_exc())
    response.set_exception(pickle.dumps(e))
    if isinstance(e, apiproxy_errors.ApplicationError):
      application_error = response.mutable_application_error()
      application_error.set_code(e.application_error)
      application_error.set_detail(e.error_detail)
  return response.Encode()

def main():
  testbed_server = Server()
  testbed_server.setUp()
  while 1:
    line = sys.stdin.readline()
    stripped = line.strip()
    logging.debug(stripped)
    
    if stripped == "#reset#":
      testbed_server.tearDown()
      testbed_server.setUp()
      
    elif stripped == "#quit#":
      return
      
    elif stripped:
      req = base64.b64decode(stripped)
      _, req = req.split("\n", 1) # discard a first row
      resp = process(req)
      resp = base64.b64encode(resp)
      sys.stdout.write(resp)
      sys.stdout.write("\n")
      sys.stdout.flush()
    
    else:
      return

if __name__ == "__main__":
  main()
