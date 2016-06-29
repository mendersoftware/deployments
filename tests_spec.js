var frisby = require('frisby');

frisby.create('Ensure test has foo and bar')
  .get(
    "http://private-f72329-deploymenttest.apiary-mock.com/api/0.0.1/deployments/?status=inprogress&name=\'Production XZY\'"
  )
  .expectJSONLength(1)
  .expectStatus(200)
  .expectJSONTypes('*', {
    id: String,
    finished: String,
    status: "pending",
    created: String,
    name: String,
    version: String
  })
  .toss()
