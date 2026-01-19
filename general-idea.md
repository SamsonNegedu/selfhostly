I'm selfhosting different apps on my raspberry pi.
For each app, I create a directory which contains a docker-compose.yml and a .env file.
The .env file contains the environment variables for the app and a TUNNEL_TOKEN variable.for cloudflared tunnel.

The docker-compose.yml file contains dedicated service for cloudflared tunnel and each app has its own tunnel.
I create the tunnel token through the following steps:
1. Login to cloudflare
2. Create a new tunnel in the Connectors page
3. Copy the tunnel token and paste it into the .env file of the app.
4. Run the docker-compose up -d command to start the app.
5. Create a new published application route in "Published application routes" section of the new tunnel

If everything works fine, I can access the app from the public URL specified in the "Published application routes" section.

I want to automate all of the above steps as best as possible possibly through a well-polished UI and possible eliminate the need to SSH into the raspberry pi

IDEA 2

For apps that are already running, when a new docker image is released, I want to be able to update the app without stopping the service.
Usually, I would ssh into the service and trigger a docker compose command
- Ideally, a button from the web UI should take care of this and show me the progress of the update.
