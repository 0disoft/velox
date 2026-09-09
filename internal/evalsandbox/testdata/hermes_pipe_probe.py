"""Diagnostic Hermes provider-client bridge; not a qualifying agent trial."""

import json
import os
import sys

# -I isolates Python startup; only the staged helper and explicit SDK root enter.
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(1, os.environ["VELOX_HERMES_SITE_PACKAGES"])

import httpx

MAX_REQUEST = 8192
MAX_RESPONSE = 65536
MODEL = "commandcode/meta-muse-spark-1.3-contributor"


class PipeTransport(httpx.BaseTransport):
    def __init__(self, reader, writer):
        self.reader = reader
        self.writer = writer
        self.used = False

    def handle_request(self, request):
        if self.used:
            raise httpx.TransportError("probe request budget exhausted")
        self.used = True
        if request.method != "POST" or str(request.url) != "http://velox-pipe.invalid/v1/chat/completions":
            raise httpx.TransportError("probe endpoint denied")
        body = request.read()
        if len(body) > MAX_REQUEST:
            raise httpx.TransportError("probe request too large")
        frame = json.dumps({"method": request.method, "path": request.url.path,
                            "body": json.loads(body)}, separators=(",", ":")).encode() + b"\n"
        if len(frame) > MAX_REQUEST:
            raise httpx.TransportError("probe frame too large")
        self.writer.write(frame)
        self.writer.flush()
        response = self.reader.readline(MAX_RESPONSE + 1)
        if not response.endswith(b"\n") or len(response) > MAX_RESPONSE:
            raise httpx.TransportError("invalid probe response frame")
        envelope = json.loads(response)
        if set(envelope) != {"status", "body"} or envelope["status"] != 200:
            raise httpx.TransportError("probe broker denied request")
        return httpx.Response(200, json=envelope["body"], request=request)

    def close(self):
        self.reader.close()
        self.writer.close()


def main():
    import msvcrt
    from agent.process_bootstrap import OpenAI

    try:
        with open(os.environ["VELOX_BROKER_FORBIDDEN"], "rb"):
            raise RuntimeError("outside file was readable")
    except PermissionError:
        pass
    reader = os.fdopen(msvcrt.open_osfhandle(int(os.environ["VELOX_BROKER_READ"]), os.O_RDONLY | os.O_BINARY), "rb")
    writer = os.fdopen(msvcrt.open_osfhandle(int(os.environ["VELOX_BROKER_WRITE"]), os.O_WRONLY | os.O_BINARY), "wb")
    transport = PipeTransport(reader, writer)
    # Empty means no credential. Headers never cross the pipe or reach the host.
    with OpenAI(api_key="", base_url="http://velox-pipe.invalid/v1",
                max_retries=0, http_client=httpx.Client(transport=transport, trust_env=False)) as client:
        response = client.chat.completions.create(
            model=MODEL, messages=[{"role": "user", "content": "Reply exactly OK. Do not use tools."}],
            stream=False, max_tokens=1024)
        if len(response.choices) != 1:
            raise RuntimeError("unexpected probe choice count")
        choice = response.choices[0]
        content = choice.message.content or ""
        if content.strip() != "OK" or choice.finish_reason != "stop" or choice.message.tool_calls:
            raise RuntimeError(f"unexpected probe completion: finish={choice.finish_reason}, chars={len(content)}, tools={bool(choice.message.tool_calls)}")
        if "muse-spark-1.3" not in response.model:
            raise RuntimeError("unexpected response-declared model")
        try:
            client.chat.completions.create(model=MODEL, messages=[])
        except httpx.TransportError:
            pass
        except Exception as error:
            # The SDK wraps transport errors without issuing a retry.
            if not isinstance(error.__cause__, httpx.TransportError):
                raise
        else:
            raise RuntimeError("second request was allowed")
    with open(os.path.join(os.environ["VELOX_BROKER_RESULT_ROOT"], "hermes-probe.json"), "x", encoding="ascii") as result:
        json.dump({"providerClient": "hermes.process_bootstrap.OpenAI", "requests": 1,
                   "secondRequestDenied": True, "outsideFileDenied": True,
                   "qualifyingTrial": False}, result)


if __name__ == "__main__":
    main()
