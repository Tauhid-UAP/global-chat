# Golang Web App

## Overview

This is a Golang web application that uses:

- **PostgreSQL** for persistent storage
- **Redis** for session management
- **AWS S3** for media storage
- **CloudFront (optional)** for CDN-backed media delivery
- **golang-migrate** for database migrations

The application can be run locally using **Docker Compose** and deployed to production as a **single Go executable**, with migrations executed separately.

---

## Running the App Locally

### Prerequisites

- Go 1.20+  
- Docker & Docker Compose  
- AWS CLI  
- Make  

---

### Local Setup Instructions

1. **Generate an application secret key**

```bash
openssl rand -hex 32
```

```Sample Output
e7ca0377e91a43a5b0a2b30978339e11cbcb1dd4578da7a00fad89bf085acd3b
```

Set it as an environment variable:

```env
APP_SECRET=<generated_secret>
```

2. **Configure AWS S3 and CloudFront (optional)**

- Create an S3 bucket for storing profile images.  
- (Optional) Set up a CloudFront distribution with read access to the bucket.  
- Ensure you have an IAM user with `s3:PutObject` permission.

Set the environment variables:

```env
AWS_REGION=<aws_region>
AWS_S3_BUCKET=<s3_bucket_name>
MEDIA_BASE_URL=<https://cloudfront_or_s3_base_url>
```

> `MEDIA_BASE_URL` must include `https://`.

3. **Install and configure AWS CLI**

```bash
aws configure
```

Verify credentials:

```bash
aws sts get-caller-identity
```

A successful response includes `UserId`, `Account`, `Arn`.

4. **Start all services**

```bash
docker compose up
```

5. **Run database migrations**

```bash
make migrate-up
```

This step does the following
- Apply the migrations.
- Remove the container of the migrate service since it is no longer needed.

6. **Verify the application**

Visit in a browser:

- http://127.0.0.1:8000/register  
- http://127.0.0.1:8000/login

---

## Deployment to Production

### One-Time Infrastructure Setup

1. **PostgreSQL**

Provision a database and set:

```env
DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=disable
```

2. **Redis**

Provision Redis and set:

```env
REDIS_ADDR=<host:port>
REDIS_PASSWORD=<password>
REDIS_DB=<db_number>
SESSION_TTL_HOURS=<hours>
```

3. **AWS S3 & CloudFront**

- Create an S3 bucket for media storage.  
- (Optional) Configure CloudFront distribution for read access.  
- Ensure IAM user has `s3:PutObject` permission.  
- Configure AWS CLI on production:

```bash
aws configure
```

---

### Deployment Steps

1. **Transfer migration files to the server**

```bash
scp -r migrations user@server:/opt/myapp/
```

2. **Run database migrations**

```bash
migrate -path /opt/myapp/migrations -database "$DATABASE_URL" up
```

3. **Set AWS-related environment variables**

```env
AWS_REGION=<production_region>
AWS_S3_BUCKET=<production_bucket>
MEDIA_BASE_URL=<https://cloudfront_or_s3_base_url>
```

4. **Build the Go executable**

```bash
go build -o myapp ./cmd/web
```

5. **Transfer the binary to the server**

```bash
scp myapp user@server:/opt/myapp/
```

6. **Run the application**

```bash
./myapp
```

7. **Verify deployment**

Check all API endpoints and web routes to ensure they work.

---

## Notes & Best Practices

- Migrations are **not run automatically**; run them before starting the app.  
- Media URLs are stored as **absolute HTTPS URLs**.  
- AWS credentials are resolved via the standard AWS chain (`~/.aws/credentials`, IAM roles).  
- The app binary is stateless and safe for horizontal scaling.  

---