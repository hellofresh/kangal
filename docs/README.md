# Kangal

Welcome to the Kangal - Kubernetes and Go Automatic Loader!
To start using Kangal you will need to install it in your cluster. Helm installation guide is [here](https://github.com/hellofresh/kangal/blob/master/charts/kangal/README.md).
In this section you can find information about load generator and how to write tests.

    
### How is load generated?
Currently the main load generator used in Kangal is JMeter v5.0 r1840935. JMeter is a powerfull tool which can be used for different performance testing tasks. 
Please read [Load generator in Kangal](Load-generator-in-kangal.md) for further details.

### Usage examples
#### Tests with test data
Some test scenarios require unique request or at least some amount of varied data in requests. For this purposes JMeter allows you to use external data sets in a CSV format. Read more about [CSV DataSetConfig](https://jmeter.apache.org/usermanual/component_reference.html#CSV_Data_Set_Config) in official JMeter documentation.

1. Prepare your test data in CSV file
2. Configure test script accordingly. Find detais here [How to write and understand a JMeter test: Test with CSV Data](How-to-write-tests.md#test-with-csv-data)
3. Add both files in POST request to Kangal API

Kangal will split the test data equally between all the distributed pods you requested, so every pod will have a unique piece of your testdata file and requests from different pods will not be duplicated. If you have only one distributed pod no data splitting will take place.

#### Tests with environment variables
Some tests may contain sensitive information like DB connection parameters, authorisation tokens, etc. You can provide this information as environment variables which will be applied in loadtest environment before running test. 

Kangal allows you to use a file with env vars saved in CSV format. Please configure your test script accordingly to use env vars. Read more about using env vars in official [JMeter-plugin documentation](https://jmeter-plugins.org/wiki/Functions/#envsupfont-color-gray-size-1-since-1-2-0-font-sup) and [How to write and understand a JMeter test](How-to-write-tests.md).

1. Save your environment variables in CSV file
2. Configure test script accordingly. Find detais here [How to write and understand a JMeter test: Test with environment variables](How-to-write-tests.md#test-with-environment-variables)
3. Add both files in POST request to Kangal API