Overview

Prerequisites:
- Docker should be installed and up to date
- Java 11 should be available at /Library/Java/JavaVirtualMachines/temurin-11.jdk/Contents/Home/

Troubleshooting

Startup steps

- Lemans
  - `make start-lemans`
  - The lemans gateway and resource server start at the same time but the gateway requires the resource server 
  to be running to get metadata about the configuration. This will result in some errors on startup for the gateway 
  until the resource server is fully up and running.
- Create the lemans stream
  - `STREAM_NAME=<stream_name> make create-buffered-stream`
- WF Proxy
  - `make start-wf-proxy`
- Sending Data
  - `make post-to-proxy`
