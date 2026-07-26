import type { NextFunction, Request, Response } from "express";

import { MicroscopeClient } from "./index.js";

export interface MicroscopeExpressOptions {
  baseUrl: string;
}

export function microscopeMiddleware(options: MicroscopeExpressOptions) {
  const client = new MicroscopeClient({ baseUrl: options.baseUrl });

  return (req: Request, res: Response, next: NextFunction): void => {
    const started = Date.now();
    res.on("finish", () => {
      void client.record("http_request", {
        method: req.method,
        path: req.path,
        status: res.statusCode,
        duration_ms: Date.now() - started,
      });
    });
    next();
  };
}
