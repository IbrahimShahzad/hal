# HAL

<p style="float:right; margin:0 0 1em 1em; width:220px;">
  <img src="images/hal.jpg" alt="HAL-9000" style="width:100%; height:auto; display:block; border-radius:4px;" />
</p>

Welcome aboard the *"Discovery One"*. Our mission is to travel to Jupiter to investigate a mysterious, signal-emitting monolith found on the moon. Our onboard Heuristically programmed ALgorithmic computer (HAL), has read the mission files and is ready to support you on your journey. Hal will ensure the mission has the highest priority.

You can log into communications channel to live stream events and chat with your fellow mission critical crew members.

Please read more about the [Discovery One](https://grokipedia.com/page/Discovery_One) here.

## BOOT-UP

![HAL Demo](images/hal-demo.gif)

*HAL boot sequence*

![HAL Demo](images/comms.png)

*HAL comms monitoring stream*

## Usage

To run the HAL-9000:

```sh
go build . -o hal
./hal -addr ":8080"
```

Then visit http://localhost:8080/ to see the HAL interface.

## Authentication

HAL uses **per-user authentication tokens**. Each registered crew member receives a unique token when they register. So fun.

## Crew Management

Crew members should have clearance to use the communication channel. Each registered member gets a unique authentication token.

### Registering Crew Members For Communication Channel

You can either use the curl or the client.

```sh
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username": "alex"}'

# Response:
# {"id":1,"username":"ALEX","token":"a1b2c3d4e5f6..."}
```

### Sending Messages

Use the token you received when registering to authenticate your messages:

```sh
curl -X POST http://localhost:8080/update \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: a1b2c3d4e5f6..." \
  -d '{"message": "Life support systems nominal", "tags": ["systems", "status"]}'
```

### Tag Processing

Crew members are encouraged to use tags. Due to human error which is always imminent, HAL converts the error prone ramblings to neat and tidy tags by
- Converting to **UPPERCASE**
- Replacing spaces with **underscores**
- Storing as Comma-separated values
- Crew rambling becomes `["work project", "urgent"]` becomes `["WORK_PROJECT", "URGENT"]`
This is necessary so that HAL can easily analyze the crew communication.

### Viewing Crew-Specific Messages

- **All users**: http://localhost:8080/
- **Specific user**: http://localhost:8080/user/alex (serves the full HAL interface filtered for alex)

## Client

For registering crew and sending messages to on the communications channel see the [client documentation](tools/client/README.md).

## CAUTION

Please, Keep someone near the pod bay doors. HAL-9000 has shown tendency to lock them from inside and not open it.

