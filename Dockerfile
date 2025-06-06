FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY alert_history_service.py ./
COPY templates ./templates

EXPOSE 8080

CMD ["uvicorn", "alert_history_service:app", "--host", "0.0.0.0", "--port", "8080"]
