import { HttpInterceptorFn } from '@angular/common/http';

export const signingInterceptorTsInterceptor: HttpInterceptorFn = (req, next) => {
  const timestamp = Date.now().toString();
  const signature = 'test-signature';
  // const signature = crypto.subtle
  // .digest('SHA-256', new TextEncoder().encode(timestamp))
  // .then((hash) => {
  //   return btoa(String.fromCharCode(...new Uint8Array(hash)));
  // });

  const signedRequest = req.clone({
    setHeaders: {
      'X-Zen-Signature': signature,
      'X-Zen-Timestamp': timestamp,
    },
  });

  return next(signedRequest);
};
