````md
# Let’s Encrypt (Certbot) + Cloudflare DNS-01 (Docker)

This repo uses **Let’s Encrypt** certificates issued via **Certbot** with the **Cloudflare DNS-01 challenge**.
DNS-01 means Certbot proves domain ownership by creating temporary TXT records in Cloudflare (no HTTP challenge needed).

Domains:
- Web: `nubecita.casa`, `www.nubecita.casa`
- MQTT: `mqtt.nubecita.casa`

---

## Prerequisites

### 1) Cloudflare API token
Create an API token with:
- Zone → DNS → Edit
- (Optional) Zone → Zone → Read
Scope it to the `nubecita.casa` zone.

Create the token file:

```bash
mkdir -p letsencrypt letsencrypt-lib secrets
chmod 700 secrets
nano secrets/cf.ini
chmod 600 secrets/cf.ini
````

`secrets/cf.ini`:

```ini
dns_cloudflare_api_token = YOUR_TOKEN_HERE
```

### 2) DNS records (Cloudflare)

* Web (`nubecita.casa` + `www`) can stay **Proxied (orange cloud)**.
* MQTT (`mqtt.nubecita.casa`) should be **DNS only (grey cloud)** and point to your server IP.

---

## Issue certificates (first time)

### Web cert: nubecita.casa + [www.nubecita.casa](http://www.nubecita.casa)

```bash
docker run --rm \
  -v "$(pwd)/letsencrypt:/etc/letsencrypt" \
  -v "$(pwd)/letsencrypt-lib:/var/lib/letsencrypt" \
  -v "$(pwd)/secrets/cf.ini:/cf.ini:ro" \
  certbot/dns-cloudflare certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /cf.ini \
  --dns-cloudflare-propagation-seconds 60 \
  -d nubecita.casa -d www.nubecita.casa \
  --agree-tos -m walen.piotr@gmail.com --non-interactive -v
```

### MQTT cert: mqtt.nubecita.casa

```bash
docker run --rm \
  -v "$(pwd)/letsencrypt:/etc/letsencrypt" \
  -v "$(pwd)/letsencrypt-lib:/var/lib/letsencrypt" \
  -v "$(pwd)/secrets/cf.ini:/cf.ini:ro" \
  certbot/dns-cloudflare certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials /cf.ini \
  --dns-cloudflare-propagation-seconds 60 \
  -d mqtt.nubecita.casa \
  --agree-tos -m walen.piotr@gmail.com --non-interactive -v
```

After issuing, you should have:

* `./letsencrypt/live/nubecita.casa/`
* `./letsencrypt/live/mqtt.nubecita.casa/`

---

## Renew certificates (manual)

Let’s Encrypt certs are valid for ~90 days. `renew` only renews when needed (typically when <30 days left).

```bash
docker run --rm \
  -v "$(pwd)/letsencrypt:/etc/letsencrypt" \
  -v "$(pwd)/letsencrypt-lib:/var/lib/letsencrypt" \
  -v "$(pwd)/secrets/cf.ini:/cf.ini:ro" \
  certbot/dns-cloudflare renew \
  --dns-cloudflare \
  --dns-cloudflare-credentials /cf.ini \
  --dns-cloudflare-propagation-seconds 60
```

---

## Reload services after renewal

After renewal, reload/restart containers so they pick up the new cert files.

### Nginx (reload)

```bash
docker compose -f docker-compose.remote.yaml exec -T nginx nginx -s reload
```

### Mosquitto (restart)

```bash
docker compose -f docker-compose.remote.yaml restart mosquitto
```

---

## Optional: check certificate expiration

Web:

```bash
openssl x509 -enddate -noout -in ./letsencrypt/live/nubecita.casa/fullchain.pem
```

MQTT:

```bash
openssl x509 -enddate -noout -in ./letsencrypt/live/mqtt.nubecita.casa/fullchain.pem
```

```
::contentReference[oaicite:0]{index=0}
```
