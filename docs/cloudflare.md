# Cloudflare + Hetzner: SSL setup (Nginx in Docker Compose)

This guide documents how to:
- point a Cloudflare-managed domain to a Hetzner server
- enable HTTPS using a **Cloudflare Origin Certificate**
- optionally restrict the origin so only **Cloudflare** can reach it

---

## Prerequisites

- A domain added to Cloudflare (nameservers already changed to Cloudflare)
- A Hetzner server with a public IPv4 (and optional IPv6)
- Docker + Docker Compose installed on the server
- An Nginx container serving a simple site (e.g. `./html/index.html`)
- You can SSH into the server

---

## 1) Cloudflare DNS records

In Cloudflare → **DNS**:

1. Create an **A** record:
   - **Name**: `@`
   - **Content**: `<YOUR_HETZNER_IPV4>`
   - **Proxy status**: ✅ **Proxied** (orange cloud)
   - TTL: Auto

2. Create `www` (recommended):
   - **CNAME**
   - **Name**: `www`
   - **Target**: `@`
   - **Proxy status**: ✅ Proxied

3. (Optional) Add IPv6:
   - **AAAA**
   - **Name**: `@`
   - **Content**: `<YOUR_HETZNER_IPV6>`
   - **Proxy status**: ✅ Proxied

> If you want only a subdomain like `app.example.com`, create an **A** record for `app` instead of `@`.

---

## 2) Create a Cloudflare Origin Certificate

In Cloudflare → **SSL/TLS** → **Origin Server** → **Create certificate**:

- **Hostnames**: add your domain(s), e.g.:
  - `example.com`
  - `www.example.com`
- Key type: RSA (default) is fine
- Save the generated:
  - **Origin Certificate** (PEM)
  - **Private Key** (KEY)

On the server, create a folder and save the files:

```bash
mkdir -p ./cf-origin
nano ./cf-origin/origin.pem   # paste the Origin Certificate
nano ./cf-origin/origin.key   # paste the Private Key
chmod 600 ./cf-origin/origin.key
