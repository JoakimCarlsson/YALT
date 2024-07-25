# YALT
###### Yet Another Load Testing tool
Inspired by K6, but with fewer features and much worse.

## Features

- Define stages to control the number of virtual users (VUs) over time
- Set thresholds for request duration percentiles and failure rates
- Simple configuration and execution

## Configuration

The load test configuration is defined in a JavaScript file. Below is an example configuration:

```javascript
exports.options = {
  thresholds: {
    // Thresholds for HTTP request durations
    // 'p(50) < 100' means 50th percentile (median) duration should be less than 100ms
    // 'p(90) < 150' means 90th percentile duration should be less than 150ms
    // 'p(95) < 200' means 95th percentile duration should be less than 200ms
    // 'p(99) < 300' means 99th percentile duration should be less than 300ms
    // 'min < 50' means minimum duration should be less than 50ms
    // 'max < 500' means maximum duration should be less than 500ms
    http_req_duration: ['p(50) < 100', 'p(90) < 150', 'p(95) < 200', 'p(99) < 300', 'min < 50', 'max < 500'],
    // Threshold for HTTP request failure rate
    // 'rate < 0.01' means failure rate should be less than 1%
    http_req_failed: ['rate < 0.01']
  },
  stages: [
    // Stages for load testing
    // Each stage defines a duration and a target number of virtual users (VUs)
    // The rampUp and rampDown properties indicate the duration to gradually add or remove VUs
    { duration: '60s', target: 100, rampUp: '30s', rampDown: '30s' },
    { duration: '1m', target: 200, rampUp: '20s', rampDown: '10s' },
    { duration: '30s', target: 50, rampUp: '5s', rampDown: '5s' },
  ],
};

// Example object to be sent in the request body (if needed)
const car = {
  make: 'Volvo',
  model: 'V50'
}

// Load test function to be executed by each virtual user
exports.loadTest = async function (client) {
  const config = {
    method: 'GET',  // HTTP method
    url: 'https://app-stresstest-lab.azurewebsites.net/hello-world',  // Target URL
    headers: {
      'Content-Type': 'application/json'  // Request headers
    }
    // Uncomment the following line to include the 'car' object in the request body
    // body: JSON.stringify(car)
  };

  // Perform the HTTP request
  await client.fetch(config);
};

```

# Explanation

### Thresholds:
- Defines performance criteria for the test.
- Example: `p(50) < 100` ensures that the median request duration is less than 100ms.
- `http_req_failed: ['rate < 0.01']` ensures that the failure rate is less than 1%.

### Stages:
- Defines the number of virtual users (VUs) and the duration for each stage.
- Example stages:
    - 30 VUs for 5 seconds.
    - 15 VUs for 5 seconds.
    - 50 VUs for 5 seconds.
- Ramp-up and Ramp-down:
    - Ramp-up: The period during which the number of virtual users (VUs) is gradually increased until it reaches the target.
    - Ramp-down: The period during which the number of virtual users (VUs) is gradually decreased to zero.
    - Example: If the stage duration is 100 seconds, with a ramp-up of 20 seconds and a ramp-down of 20 seconds:
        - The test will ramp up for 20 seconds, gradually increasing the VUs to the target.
        - Then it will run at the target number of VUs for the stable period of 60 seconds (100s - 20s ramp-up - 20s ramp-down).
        - Finally, it will ramp down for 20 seconds, gradually decreasing the VUs to zero.

### Load Test Function:
- Defines the actions performed by each virtual user during the test.
- Example: Sends a GET request to the specified URL.
- Optionally, includes a JSON object in the request body.