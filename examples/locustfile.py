import glob, tarfile, requests, os

from locust import HttpUser, task, between, events, runners
from locust.runners import MasterRunner

@events.quitting.add_listener
def hook_quit(environment):
    presigned_url = os.environ.get('REPORT_PRESIGNED_URL')
    if None == presigned_url:
        return
    if False == isinstance(environment.runner, MasterRunner):
        return
    report = '/home/locust/report.tar.gz'
    tar = tarfile.open(report, 'w:gz')
    for item in glob.glob('/tmp/*.csv'):
        print('Adding %s...' % item)
        tar.add(item, arcname=os.path.basename(item))
    tar.close()
    request_headers = {'content-type': 'application/gzip'}
    r = requests.put(presigned_url, data=open(report, 'rb'), headers=request_headers)

class ExampleLoadTest(HttpUser):
    wait_time = between(5, 15)

    @task
    def example_page(self):
        self.client.get('/example-page')
