# go 스터디 위한 RAG 기반 Chatbot API 프로젝트 
### 1. 개요
챗봇 질의를 위한 API  서버 구축

- RAG 위한 벡터 데이터 저장 API
- CHAT API

  - 질의 문을 vector 처리 하여 vector db 조회
  - documents 들의 rerank 처리
  - 최종 documents들과 질의문을 llm 요청

### 2. Todo
    
### 3. ETC

- google colab에 ollama 이용

## script

# Ollama 설치
!curl -fsSl https://ollama.com/install.sh | sh

%%bash
export OLLAMA_HOST=0.0.0.0:11434
nohup ollama serve > ollama.log 2>&1 &

!ollama pull gemma3:4b

!ollama pull bge-m3

!ollama pull qwen2.5:3b

%%bash
cat <<EOF > llama3-3b-rerank.modelfile
FROM llama3.2:3b
PARAMETER temperature 0
PARAMETER top_p 1
PARAMETER num_predict 80
EOF

!ollama create llama3-3b-rerank -f llama3-3b-rerank.modelfile

!ollama list

# ngrok 설치

!wget https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-stable-linux-amd64.zip

!unzip ngrok-stable-linux-amd64.zip

!./ngrok config add-authtoken {token}

!pkill ngrok

!./ngrok http 11434 --log=stdout > ngrok.log 2>&1 &
!sleep 3

!grep -o 'https://[^ ]*' ngrok.log

!ps aux | grep ollama

