import { defineConfig } from "cypress";
// @ts-expect-error
import { removeDirectory } from "cypress-delete-downloads-folder";
import { spawn } from "node:child_process";

module.exports = defineConfig({
  e2e: {
    baseUrl: "http://localhost:8080",
    setupNodeEvents(on, config) {
      on("task", {
        "db:seed": async (scenarioName) => {
          return new Promise((resolve, reject) => {
            var args = ["run", "./seed", scenarioName];
            const seedProcess = spawn("go", args);

            seedProcess.stdout.on("data", (data) => {
              console.log(`stdout: ${data}`);
            });

            seedProcess.stderr.on("data", (data) => {
              console.error(`db:seed: ${data}`);
            });

            seedProcess.on("close", (code) => {
              if (code === 0) {
                resolve(true);
              } else {
                reject(`db:seed child process exited with code ${code}`);
              }
            });
          });
        },
        removeDirectory,
      });
    },
  },
});
