# Image Processing API

This project is an Image Processing API developed in Golang, designed to handle image uploads to an S3 compatible storage (MinIO), caching with Redis, and basic retrieval and updating of images. The API supports three main operations: GET, POST, and PUT.

## Application Setup

The application is designed to be run using Docker and Docker Compose, with services including MinIO, Redis, Prometheus, Grafana, and Nginx.

### Challenge

Your challenge, should you choose to accept, is to write the `docker-compose.yml` from scratch based on the provided application code and the following specifications:

#### Services:

1. **App Service**:
    - Written in Golang.
    - Exposes and listens on port 8080.
    - Depends on MinIO and Redis services.
    - Requires several environment variables to interact with MinIO and Redis:
        - `AWS_ACCESS_KEY_ID=minioadmin`
        - `AWS_SECRET_ACCESS_KEY=minioadmin`
        - `AWS_REGION=us-east-1`
        - `S3_BUCKET=mybucket`
        - `S3_ENDPOINT=minio:9000`
        - `REDIS_ADDRESS=redis:6379`

2. **MinIO (S3 Storage)**:
    - Should operate on standard MinIO ports.
    - Requires setting up access keys and a default bucket.

3. **Redis**:
    - Standard Redis setup to be used for caching.

4. **Nginx**:
    - Should serve static content from a test folder that includes an HTML file and images (images are not provided, you can use any images you want).

5. **Prometheus and Grafana**:
    - Set up for monitoring the application.

### Requirements:

- The `docker-compose.yml` file must correctly establish the network connections between services.
- Include volume management for persistent data storage where appropriate.
- Ensure the application is secure and robust against common failures.

### Testing:

Once your Docker Compose is set up, perform the following actions to test its functionality:

1. Access the static page served by Nginx.
2. Use the API to upload images to MinIO.
3. Retrieve and update images using the API.

## Contribution

Fork this repository, create your Docker Compose file, and submit a pull request with your additions. Your PR should include a brief explanation of your Docker setup, smoke test scenarios and any assumptions you've made during the challenge.

Good luck!