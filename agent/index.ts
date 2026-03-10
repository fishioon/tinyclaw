import Redis from "ioredis";
import { query } from "@anthropic-ai/claude-code";

const STREAM = process.env.STREAM_KEY ?? "stream:group:default";
const GROUP = "tinyclaw";
const CONSUMER = `agent-${process.pid}`;
const BLOCK_MS = 5000;

const redis = new Redis(process.env.REDIS_URL ?? "redis://localhost:6379");

async function ensureGroup() {
  try {
    await redis.xgroup("CREATE", STREAM, GROUP, "0", "MKSTREAM");
  } catch (e: any) {
    if (!e.message?.includes("BUSYGROUP")) throw e;
  }
}

async function process(id: string, raw: string) {
  let output = "";
  for await (const chunk of query({ prompt: raw, options: { maxTurns: 10 } })) {
    if (chunk.type === "result") {
      output = chunk.result ?? "";
    }
  }
  console.log(`processed id=${id} output_len=${output.length}`);
}

async function run() {
  await ensureGroup();
  console.log(`agent started stream=${STREAM} group=${GROUP} consumer=${CONSUMER}`);

  while (true) {
    const results = await redis.xreadgroup(
      "GROUP", GROUP, CONSUMER,
      "COUNT", "1",
      "BLOCK", String(BLOCK_MS),
      "STREAMS", STREAM, ">"
    );

    if (!results) continue;

    for (const [, messages] of results as any[]) {
      for (const [id, fields] of messages) {
        const raw = fields[fields.indexOf("raw") + 1] ?? "";
        try {
          await process(id, raw);
          await redis.xack(STREAM, GROUP, id);
        } catch (e) {
          console.error(`error id=${id}`, e);
        }
      }
    }
  }
}

run().catch(console.error);
