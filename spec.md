## Overal structure

/stutzthings

  /stutzthings-server
    - written in Golang
    - mqtt2influxdb
      - daemon in background bridging data from mqtt to influxdb
    - operations
      - GET /[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name] - query influxdb
      - PUT /[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name] - set mqtt topic so that IoT receives it
      - GET /[account_id]/[device_id]/[device_instance_id] - query influxdb for latest data on device state
      - rest endpoints are protected by checking JWT in Bearer token and verifying the scope claim, such as "i:account_id/device_id/device_instance_id:wrs" for being able to (r)ead, (w)rite attributes and (s)et desired states on [node_name]/set. scopes with glob patterns are supported also. example: "i:12345/akinator2000/*:r" gives read access to all device instances of akinator2000
      - POST /registration - creates a new device Id and generate jwt with claims wrs for mqtt - stateless
      - GET /[account_id]/[device_id]/[device_instance_id]/[node_name]/history?since=2d - query device data from influxdb
      - GET /[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]/history?since=2d - query specific attribute history from influxdb
      - expose operations via RestAPI and MCP Server
      - for most of this, migrate code from https://github.com/stutzlab/stutzthings-proxy. open all projects and look for the js code
      - `make run`: starts local mqtt mosquitto, influxdb and stutzthings-server, all connected and running

  /stutzthings-sdk-python
    - written in Python
    - SDK with MCP client that connects to stutzthings server MCP server (to be used by MCP Hosts)
    - SDK to stutzthings rest APIs
    - SDK to connect to MQTT and receive realtime events for specific devices
  
## MQTT topic spec

This spec defines the communication protocol for IoT devices to report it's internal status (sensors, actuators etc) and to receive commands (via /set)

/v1/[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name] - sensor or actuator current value

/v1/[account_id]/[device_id]/[device_instance_id]/[node_name]/[attribute_name]/set - subscribed by the IoT device to know when it has to actuate on something. after actuating, it will update the value in [attribute_name] to indicate the task was performed so that other parties can see if the state has changed. Normally used for configuration changing or actuators

/v1/[account_id]/[device_id]/[device_instance_id]/$info/$schema - json schema describing node_name/attributes

account_id - cloud account
device_id - identifies a type of device. ex: LedMatrix2000
device_instance_id - sensorator123-room, sensorator345-outside
node_name - ledmatrix16x16, lamp, temperature, mic, cam, bip_speaker
