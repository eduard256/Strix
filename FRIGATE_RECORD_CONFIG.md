# Frigate Configuration - Запись по движению с детекцией объектов

Конфигурация для Frigate с записью при обнаружении движения и детекцией объектов (person, car, cat, dog).

## Dual-Stream конфиг (Main + Sub) - РЕКОМЕНДУЕТСЯ

Используется sub stream для детекции (экономия CPU), main stream для записи (качество).

```yaml
mqtt:
  enabled: false

# Глобальные настройки записи
record:
  enabled: true
  retain:
    days: 7
    mode: motion  # Записывать только при движении

# Go2RTC Configuration (Frigate built-in)
go2rtc:
  streams:
    '10_0_20_112_main':
      - rtsp://admin:password@10.0.20.112/live/main

    '10_0_20_112_sub':
      - rtsp://admin:password@10.0.20.112/live/sub

# Frigate Camera Configuration
cameras:
  camera_10_0_20_112:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/10_0_20_112_sub
          input_args: preset-rtsp-restream
          roles:
            - detect
        - path: rtsp://127.0.0.1:8554/10_0_20_112_main
          input_args: preset-rtsp-restream
          roles:
            - record
    live:
      streams:
        Main Stream: 10_0_20_112_main    # HD для просмотра
        Sub Stream: 10_0_20_112_sub      # Низкое разрешение (опционально)
    objects:
      track:
        - person
        - car
        - cat
        - dog
    record:
      enabled: true

version: 0.16-0
```

## Single-Stream конфиг (Main только)

Когда нет sub stream - используется main для детекции и записи.

```yaml
mqtt:
  enabled: false

# Глобальные настройки записи
record:
  enabled: true
  retain:
    days: 7
    mode: motion  # Записывать только при движении

# Go2RTC Configuration (Frigate built-in)
go2rtc:
  streams:
    '10_0_20_112_main':
      - rtsp://admin:password@10.0.20.112/stream1

# Frigate Camera Configuration
cameras:
  camera_10_0_20_112:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/10_0_20_112_main
          input_args: preset-rtsp-restream
          roles:
            - detect
            - record
    objects:
      track:
        - person
        - car
        - cat
        - dog
    record:
      enabled: true

version: 0.16-0
```

## Режимы записи

### `mode: motion` (рекомендуется)
Записывает видео при обнаружении движения. Экономит место на диске.

### `mode: active_objects`
Записывает только когда обнаружены объекты (person, car, etc). Еще больше экономия.

### `mode: all`
Записывает постоянно 24/7. Требует много места на диске.

## Преимущества Dual-Stream подхода

✅ **Низкая нагрузка на CPU** - детекция на sub stream (обычно 352x288 или 640x480)
✅ **Качественная запись** - запись на main stream в полном разрешении (HD/4K)
✅ **Быстрая детекция** - меньше пикселей = быстрее обработка
✅ **Авто-определение разрешения** - Frigate сам определяет параметры потока
✅ **Одно подключение к камере** - Go2RTC мультиплексирует потоки

## Что делает этот конфиг

✅ **Детекция** - работает постоянно, ищет объекты
✅ **Запись** - начинается при движении
✅ **Объекты** - распознает person, car, cat, dog
✅ **Хранение** - 7 дней записи
✅ **Snapshots** - сохраняются автоматически при детекции

## Добавление других объектов

Чтобы добавить больше объектов для детекции, измените секцию `objects.track`:

```yaml
objects:
  track:
    - person
    - car
    - cat
    - dog
    - motorcycle  # Мотоциклы
    - bicycle     # Велосипеды
    - truck       # Грузовики
    - bus         # Автобусы
```

Полный список доступных объектов: https://docs.frigate.video/configuration/objects/

## Примечания

- Dual-stream экономит CPU, используйте когда камера поддерживает sub stream
- Single-stream проще, но требует больше CPU для детекции (особенно на 4K)
- Frigate автоматически определяет разрешение потоков, блок `detect` не нужен
- Запись по движению экономит место, но может пропустить начало события
- Для непрерывной записи используйте `mode: all`
- Frigate автоматически управляет удалением старых записей
- Main stream поддерживает любое разрешение: HD (1920x1080), 4K (3840x2160) и выше
