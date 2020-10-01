# Reporting in JMeter

- [Live metrics](#live-metrics-reporting)
- [Static report](#static-report)

## Live metrics reporting
JMeter offers a possibility to send live metrics from the running test to InfluxDB (see [details in the related JMeter doc](jmeter-in-kangal/How-to-write-tests.md#metrics-collector)).
This will help you to monitor current behaviour of your test and the service under the test. 
InfluxDB installed in your cluster is required to enable this functionality.
> Note: InfluxDB is not the part of Kangal, read more about it in [InfluxDB official documentation](https://github.com/influxdata/influxdb).

You can use InfluxDB as a datasource for Grafana and create a set of useful graphs.
<p align="center"><img src="jmeter-in-kangal/images/grafana_example.png" height="500"></p>

## Static report
JMeter also offers a functionality to generate HTML report dashboards after the end of the test. Read more about it [in official JMeter docs](https://jmeter.apache.org/usermanual/generating-dashboard.html).
[Kangal-JMeter](https://github.com/hellofresh/kangal-jmeter) docker image implements this functionality.
