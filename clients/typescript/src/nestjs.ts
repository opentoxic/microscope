import type { CallHandler, ExecutionContext, NestInterceptor } from "@nestjs/common";
import { Inject, Injectable, Module } from "@nestjs/common";
import type { DynamicModule } from "@nestjs/common";
import { Observable, tap } from "rxjs";

import { MicroscopeClient } from "./index.js";

export const MICROSCOPE_CLIENT = "MICROSCOPE_CLIENT";

export interface MicroscopeModuleOptions {
  baseUrl: string;
}

@Module({})
export class MicroscopeModule {
  static forRoot(options: MicroscopeModuleOptions): DynamicModule {
    return {
      module: MicroscopeModule,
      providers: [
        {
          provide: MICROSCOPE_CLIENT,
          useValue: new MicroscopeClient({ baseUrl: options.baseUrl }),
        },
      ],
      exports: [MICROSCOPE_CLIENT],
      global: true,
    };
  }
}

@Injectable()
export class MicroscopeInterceptor implements NestInterceptor {
  constructor(@Inject(MICROSCOPE_CLIENT) private readonly client: MicroscopeClient) {}

  intercept(context: ExecutionContext, next: CallHandler): Observable<unknown> {
    const request = context.switchToHttp().getRequest();
    const response = context.switchToHttp().getResponse();
    const started = Date.now();

    return next.handle().pipe(
      tap(() => {
        void this.client.record("http_request", {
          method: request.method,
          path: request.url,
          status: response.statusCode,
          duration_ms: Date.now() - started,
        });
      }),
    );
  }
}
