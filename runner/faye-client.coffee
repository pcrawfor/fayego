Faye = require("faye")

client = new Faye.Client('http://localhost:3000/faye')


subscription = client.subscribe "/foo", (data) ->
  console.log "GOT SOME MESSAGE: #{message}"

client.publish("/foo", {message: 'MESSAGE FROM C1!'})


subscription.callback ->
  console.log "Subscription is now active!"

subscription.errback (error) ->
  console.log "WTF!"
  console.log error.message
