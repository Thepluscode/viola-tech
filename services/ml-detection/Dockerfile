FROM python:3.12-slim AS builder
WORKDIR /app
COPY services/ml-detection/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY services/ml-detection/ .

EXPOSE 8095 9095
CMD ["python", "-m", "cmd.ml-detection.main"]
