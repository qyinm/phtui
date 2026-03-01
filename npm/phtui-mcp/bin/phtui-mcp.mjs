#!/usr/bin/env node

import { spawn } from "node:child_process";

const goTarget =
  process.env.PHTUI_MCP_GO_TARGET ??
  "github.com/qyinm/phtui/cmd/phtui-mcp-stdio@main";

const goCheck = spawn("go", ["version"], {
  stdio: ["ignore", "ignore", "ignore"],
});

goCheck.on("error", () => {
  process.stderr.write(
    "phtui-mcp requires Go to be installed. Install Go from https://go.dev/dl and retry.\n",
  );
  process.exit(1);
});

goCheck.on("close", (code) => {
  if (code !== 0) {
    process.stderr.write(
      "phtui-mcp requires Go to be installed and available in PATH.\n",
    );
    process.exit(1);
    return;
  }

  const child = spawn(
    "go",
    [
      "run",
      goTarget,
    ],
    {
      stdio: "inherit",
      env: process.env,
    },
  );

  child.on("error", (err) => {
    process.stderr.write(`failed to start phtui-mcp: ${err.message}\n`);
    process.exit(1);
  });

  child.on("exit", (exitCode, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(exitCode ?? 0);
  });
});
