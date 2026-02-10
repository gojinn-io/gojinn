import { handle, Request, Response, logger, kv } from './gojinn.js';
globalThis.Gojinn = {
    handle,
    Request,
    Response,
    logger,
    kv
};