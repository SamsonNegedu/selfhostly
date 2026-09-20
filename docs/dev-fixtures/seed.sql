INSERT OR IGNORE INTO nodes (id,name,api_endpoint,api_key,is_primary,status,last_seen) VALUES
 ('node-nas','Garage NAS','http://192.168.1.42:8080','k-nas',0,'online',datetime('now','-20 seconds')),
 ('node-vps','Hetzner VPS','http://10.8.0.3:8080','k-vps',0,'offline',datetime('now','-3 hours'));
INSERT OR IGNORE INTO apps (id,name,description,compose_content,status,error_message,node_id,public_url,tunnel_mode,created_at,updated_at) VALUES
 ('a1','nextcloud','Personal cloud storage','services:
  app:
    image: nextcloud:29
    ports:
      - "8081:80"
    volumes:
      - /data/nextcloud:/var/www/html
    environment:
      - NEXTCLOUD_ADMIN_USER=admin
','running',NULL,'primary','https://cloud.example.com','custom',datetime('now','-40 days'),datetime('now','-2 days')),
 ('a2','pihole','Network-wide ad blocking','services:
  pihole:
    image: pihole/pihole:latest
    ports:
      - "8053:80"
','running',NULL,'primary',NULL,'',datetime('now','-90 days'),datetime('now','-9 days')),
 ('a3','jellyfin','Media server','services:
  jellyfin:
    image: jellyfin/jellyfin
    ports:
      - "8096:8096"
    volumes:
      - /mnt/media:/media
','stopped',NULL,'node-nas','https://watch.example.com','custom',datetime('now','-60 days'),datetime('now','-1 days')),
 ('a4','vaultwarden','Password manager','services:
  vw:
    image: vaultwarden/server
    ports:
      - "8082:80"
','error','port is already allocated: 0.0.0.0:8082','primary',NULL,'',datetime('now','-12 days'),datetime('now','-1 hours')),
 ('a5','homepage','Dashboard','services:
  hp:
    image: ghcr.io/gethomepage/homepage
    ports:
      - "3010:3000"
','running',NULL,'node-nas','https://trycloudflare.com/quick-abc','quick',datetime('now','-5 days'),datetime('now','-5 days')),
 ('a6','gitea','Git hosting','services:
  gitea:
    image: gitea/gitea:1.22
    ports:
      - "3001:3000"
','stopped',NULL,'node-vps',NULL,'',datetime('now','-30 days'),datetime('now','-30 days'));
INSERT OR IGNORE INTO cloudflare_tunnels (id,app_id,tunnel_id,tunnel_name,tunnel_token,account_id,status,public_url,ingress_rules) VALUES
 ('t1','a1','tid-1','nextcloud-tunnel','tok','acc','active','https://cloud.example.com','[{"hostname":"cloud.example.com","service":"http://app:80"}]'),
 ('t2','a3','tid-2','jellyfin-tunnel','tok','acc','active','https://watch.example.com','[{"hostname":"watch.example.com","service":"http://jellyfin:8096"}]');
INSERT OR IGNORE INTO app_schedules (id,app_id,start_cron,stop_cron,timezone,enabled) VALUES ('s1','a3','0 18 * * *','0 23 * * *','Europe/London',1);
INSERT OR IGNORE INTO compose_versions (id,app_id,version,compose_content,change_reason,changed_by,is_current) VALUES
 ('v1','a1',1,'services:
  app:
    image: nextcloud:28
','Initial version','user',0),
 ('v2','a1',2,'services:
  app:
    image: nextcloud:29
','Upgrade to 29','user',1);
INSERT OR IGNORE INTO jobs (id,type,app_id,status,progress,progress_message,error_message,started_at,completed_at,created_at) VALUES
 ('j1','app_start','a4','failed',60,'Starting containers','port is already allocated: 0.0.0.0:8082',datetime('now','-1 hours'),datetime('now','-59 minutes'),datetime('now','-1 hours')),
 ('j2','app_start','a1','completed',100,'Done',NULL,datetime('now','-2 days'),datetime('now','-2 days'),datetime('now','-2 days'));
INSERT OR IGNORE INTO job_logs (job_id,seq,line) VALUES
 ('j1',1,'Pulling vaultwarden/server ...'),('j1',2,'Creating network vaultwarden_default'),('j1',3,'Error: port is already allocated: 0.0.0.0:8082');

-- The Go scanner reads these columns as plain strings, so NULL breaks GET /api/apps.
UPDATE apps SET
  tunnel_token = COALESCE(tunnel_token, ''),
  tunnel_id = COALESCE(tunnel_id, ''),
  tunnel_domain = COALESCE(tunnel_domain, ''),
  public_url = COALESCE(public_url, ''),
  error_message = COALESCE(error_message, ''),
  description = COALESCE(description, '');

-- Extra apps on the primary node so the stopped, scheduled and updating card states can be tested.
INSERT OR IGNORE INTO apps (id,name,description,compose_content,status,error_message,node_id,public_url,tunnel_mode,tunnel_token,tunnel_id,tunnel_domain,created_at,updated_at) VALUES
 ('a7','uptime-kuma','Status and uptime checks','services:
  kuma:
    image: louislam/uptime-kuma:1
    ports:
      - "3002:3001"
','stopped','','primary','','','','','',datetime('now','-20 days'),datetime('now','-3 days')),
 ('a8','immich','Photo and video backup','services:
  immich:
    image: ghcr.io/immich-app/immich-server:release
    ports:
      - "2283:2283"
','updating','','primary','','','','','',datetime('now','-15 days'),datetime('now','-1 hours'));
INSERT OR IGNORE INTO app_schedules (id,app_id,start_cron,stop_cron,timezone,enabled) VALUES ('s2','a7','0 8 * * *','0 22 * * *','Europe/London',1);

-- Immich uses a Quick Tunnel so the Access tab can be checked in that state.
UPDATE apps SET tunnel_mode = 'quick', public_url = 'https://quick-demo.trycloudflare.com' WHERE id = 'a8';
