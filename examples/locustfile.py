import glob
import os
import tarfile

import requests
from locust import HttpUser, between, events, task
from locust.runners import MasterRunner


@events.quitting.add_listener
def hook_quit(environment):
    presigned_url = os.environ.get("REPORT_PRESIGNED_URL")
    if presigned_url is None:
        return
    if not isinstance(environment.runner, MasterRunner):
        return
    report = "/home/locust/report.tar.gz"
    tar = tarfile.open(report, "w:gz")
    for item in glob.glob("/tmp/*.csv"):
        print("Adding %s..." % item)
        tar.add(item, arcname=os.path.basename(item))
    tar.close()
    request_headers = {"content-type": "application/gzip"}
    requests.put(presigned_url, data=open(report, "rb"), headers=request_headers)


class ExampleLoadTest(HttpUser):
    wait_time = between(5, 15)

    @task
    def example_page(self):
        self.client.get("/example-page")
