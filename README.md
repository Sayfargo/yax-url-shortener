### Сравнение профилей памяти

После оптимизации генератора коротких идентификаторов и упрощения валидации URL было выполнено сравнение heap-профилей.

#### До оптимизации
```text
ROUTINE ======================== ://github.com in C:\Users\sayf4\ya_practicum\yax-url-shortener\internal\service\service.go
   2424860   81876839 (flat, cum) 51.09% of Total
         .          .    293:   uidUUID, err := uuid.Parse(uid)
         .          .    294:   if err != nil {
         .          .    295:           return "", fmt.Errorf("uuid from bytes: %w", err)
         .          .    296:   }
         .          .    297:
         .          .    298:   if err := validateURL(url); err != nil {
         .          .    299:           return "", ErrIncorrectURL
         .          .    300:   }
         .          .    301:
         .          .    302:   for attempt := 0; attempt < 3; attempt++ {
   1310740    3748001    303:
         .          .    304:           shortCode, err := s.generator.Generate()
         .          .    305:           if err != nil {
         .          .    306:                   return "", fmt.Errorf("generator generate: %w", err)
         .          .    307:           }
         .          .    308:
         .    2625854    309:           shortenedURL := model.ShortenedURL{
         .          .    310:                   UUID:        uuid.New(),
         .          .    311:                   ShortCode:   shortCode,
         .          .    312:                   OriginalURL: url,
         .          .    313:                   UserID:      uidUUID,
         .          .    314:                   IsDeleted:   false,
         .    1343508    315:           }
         .          .    316:
         .          .    317:           err = s.repo.Create(ctx, shortenedURL)
         .          .    318:
         .          .    319:           var origURLConflictErr *urlrepo.OriginalURLConflictError
         .          .    320:
         .          .    321:           switch {
         .   71603520    322:           case err == nil:
         .          .    323:
   1114120    1114120    324:                   return s.buildShortedURL(shortCode), nil
         .          .    325:
         .          .    326:           case errors.Is(err, urlrepo.ErrConflictShortCode):
         .          .    327:
         .          .    328:                   s.log.Info(
         .    1441836    329:                           "short code collision occurred, retrying",
         .          .    330:                           "url", url,
         .          .    331:                           "code", shortCode,
         .          .    332:                           "attempt", attempt+1,
         .          .    333:                   )
         .          .    334:                   continue
(pprof) 
```

#### После оптимизации
```text
ROUTINE ======================== ://github.com in C:\Users\sayf4\ya_practicum\yax-url-shortener\internal\service\service.go
    983047   38335794 (flat, cum) 48.84% of Total
         .          .    288:func (s *URLShortenerService) CreateShortURL(ctx context.Context, url, uid string) (string, error) {
         .          .    289:   if ctx.Err() != nil {
         .          .    290:           return "", ctx.Err()
         .          .    291:   }
         .          .    292:
         .          .    293:   uidUUID, err := uuid.Parse(uid)
         .          .    294:   if err != nil {
         .          .    295:           return "", fmt.Errorf("uuid from bytes: %w", err)
         .          .    296:   }
         .          .    297:
         .     688222    298:   if err := validateURL(url); err != nil {
         .          .    299:           return "", ErrIncorrectURL
         .          .    300:   }
         .          .    301:
         .          .    302:   for attempt := 0; attempt < 3; attempt++ {
         .          .    303:
         .     294916    304:           shortCode, err := s.generator.Generate()
         .          .    305:           if err != nil {
         .          .    306:                   return "", fmt.Errorf("generator generate: %w", err)
         .          .    307:           }
         .          .    308:
         .          .    309:           shortenedURL := model.ShortenedURL{
         .     622601    310:                   UUID:        uuid.New(),
         .          .    311:                   ShortCode:   shortCode,
         .          .    312:                   OriginalURL: url,
         .          .    313:                   UserID:      uidUUID,
         .          .    314:                   IsDeleted:   false,
         .          .    315:           }
         .          .    316:
         .   35009706    317:           err = s.repo.Create(ctx, shortenedURL)
         .          .    318:
    983047     983047    319:           var origURLConflictErr *urlrepo.OriginalURLConflictError
         .          .    320:
         .          .    321:           switch {
         .          .    322:           case err == nil:
         .          .    323:
         .     737302    324:                   return s.buildShortedURL(shortCode), nil
         .          .    325:
         .          .    326:           case errors.Is(err, urlrepo.ErrConflictShortCode):
         .          .    327:
         .          .    328:                   s.log.Info(
         .          .    329:                           "short code collision occurred, retrying",
```

#### Вывод pprof diff
```text
Showing nodes accounting for -6161.39kB, 44.46% of 13859.39kB total
Dropped 5 nodes (cum <= 69.30kB)
      flat  flat%   sum%        cum   cum%
-4102.80kB 29.60% 29.60% -4102.80kB 29.60%  runtime.mallocgc
   -1026kB  7.40% 37.01%    -1026kB  7.40%  bufio.NewWriterSize (inline)
 1024.22kB  7.39% 29.62%  1024.22kB  7.39%  net.newFD (inline)
 -525.43kB  3.79% 33.41%  -525.43kB  3.79%  ://github.com (inline)
  516.64kB  3.73% 29.68%   516.64kB  3.73%  net/http.(*Server).trackConn
  513.31kB  3.70% 25.98%  -524.61kB  3.79%  ://github.com
 -512.50kB  3.70% 29.67% -1037.93kB  7.49%  ://github.com
 -512.50kB  3.70% 33.37%  -512.50kB  3.70%  log/slog/internal/buffer.init.func1
 -512.16kB  3.70% 37.07%  -512.01kB  3.69%  net/http.readRequest
 -512.12kB  3.70% 40.76%  -512.12kB  3.70%  internal/bytealg.MakeNoZero
 -512.10kB  3.69% 44.46%  -512.10kB  3.69%  context.(*cancelCtx).propagateCancel
  512.05kB  3.69% 40.76%   512.05kB  3.69%  sync.runtime_notifyListWait
 -512.01kB  3.69% 44.46%  -512.01kB  3.69%  net/textproto.MIMEHeader.Add (inline)
         0     0% 44.46%  -512.10kB  3.69%  context.AfterFunc
         0     0% 44.46% -1536.23kB 11.08%  ://github.com
         0     0% 44.46% -2048.73kB 14.78%  ://github.com
         0     0% 44.46% -1536.23kB 11.08%  ://github.com
         0     0% 44.46%  1540.86kB 11.12%  ://github.com
         0     0% 44.46%  -512.10kB  3.69%  ://github.com
         0     0% 44.46%  -512.10kB  3.69%  ://github.com
         0     0% 44.46%  -512.10kB  3.69%  ://github.com
         0     0% 44.46% -1536.23kB 11.08%  ://github.com
         0     0% 44.46% -2048.73kB 14.78%  ://github.com
         0     0% 44.46% -1536.23kB 11.08%  ://github.com
         0     0% 44.46%  -512.10kB  3.69%  ://github.com
         0     0% 44.46%  -512.10kB  3.69%  ://github.comRow (inline)
```