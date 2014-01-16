faye = require 'faye'

client = new faye.Client('http://localhost:3000/faye')

subscription = client.subscribe("/foo", (message) ->
  console.log "C2 Received Message: #{message}"
)

client.publish("/foo", {message: 'Message From C2'})

subscription.callback ->
  console.log "Subscription is now active!"
  client.publish("/foo", {message: 'C2 LIVES!'})

subscription.errback (error) ->
  console.log "WTF!"
  console.log error.message
