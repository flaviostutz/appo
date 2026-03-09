# devices

Hardware and firmware — one sub-directory per device type.

- **Physical devices** (e.g. `esp32-cam-mic-matrix/`): schematics (KiCad), PCB layout files, and firmware source (PlatformIO/C++).
- **Virtual devices** (e.g. `alexa/`): a Python integration library implementing the stutzthings MQTT protocol.

Directory name convention: `<platform>-<feature-list>/` for physical boards, `<service-name>/` for virtual integrations.

## Architecture overview

Each device sub-directory is self-contained: it holds schematics, firmware, and any integration library. Physical devices communicate over MQTT using the topic schema defined by `stutzthings/`. Virtual devices wrap a third-party service in the same protocol.

## How to build

```bash
make build
```

For firmware (PlatformIO), enter the device sub-directory and follow its own README.

## How to run

Physical devices: flash firmware to the board, then power it on. Virtual devices: run via the device sub-directory instructions.
