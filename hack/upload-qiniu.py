#!/usr/bin/env python

# Copyright 2020 TiKV Project Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

import sys
import os
from qiniu import Auth, put_file, etag, urlsafe_base64_encode
import qiniu.config
from qiniu.compat import is_py2, is_py3


ACCESS_KEY = os.getenv('QINIU_ACCESS_KEY')
SECRET_KEY = os.getenv('QINIU_SECRET_KEY')
BUCKET_NAME = os.getenv('QINIU_BUCKET_NAME')

assert(ACCESS_KEY and SECRET_KEY and BUCKET_NAME)

def progress_handler(progress, total):
    print("{}/{} {:.2f}".format(progress, total, progress/total*100))

def upload(local_file, remote_name, ttl=3600):
    print(local_file, remote_name, ttl)
    q = Auth(ACCESS_KEY, SECRET_KEY)

    token = q.upload_token(BUCKET_NAME, remote_name, ttl)

    ret, info = put_file(token, remote_name, local_file)
    print("ret", ret)
    print("info", info)
    # assert ret['key'] == remote_name
    if is_py2:
      assert ret['key'].encode('utf-8') == remote_name
    elif is_py3:
      assert ret['key'] == remote_name

    assert ret['hash'] == etag(local_file)

if __name__ == "__main__":
    local_file = sys.argv[1]
    remote_name = sys.argv[2]
    upload(local_file, remote_name)

    print("https://charts.pingcap.org/{}".format(remote_name))
