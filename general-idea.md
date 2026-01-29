I'm selfhosting different apps on my Raspberry Pi, VPS, etc.
For each app, I create a directory which contains a docker-compose.yml and a .env file.
The .env file contains the environment variables for the app and a TUNNEL_TOKEN variable for the cloudflared tunnel.

The docker-compose.yml file contains dedicated service for cloudflared tunnel and each app has its own tunnel.
I create the tunnel token through the following steps:
1. Login to cloudflare
2. Create a new tunnel in the Connectors page
3. Copy the tunnel token and paste it into the .env file of the app.
4. Run the docker-compose up -d command to start the app.
5. Create a new published application route in "Published application routes" section of the new tunnel
If everything works fine, I can access the app from the public URL specified in the "Published application routes" section.

IDEA 1
I want to automate all of the above steps as best as possible through a well-polished UI and eliminate the need to SSH into my node(Raspberry Pi, Ubuntu, etc.) or Cloudflare to spin up a docker-compose-based service and create a tunnnel connected to a domain.

IDEA 2
For apps that are already running and created to the app-creation step, when a new docker image is released, I want to be able to update the app without an SSH step.
typically, this would mean:
1. SSH into the node
2. Trigger a docker compose command to pull the latest image
3. Trigger a docker compose command to recreate the service
   
Ideally, I want to be able to click a button from the web UI to trigger the update/restart of the service
