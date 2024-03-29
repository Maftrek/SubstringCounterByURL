Программа читает из stdin строки, содержащие URL.

На каждый URL отправляет HTTP-запрос методом GET и считает кол-во вхождений строки "Go" в теле ответа.

В конце работы приложение выводит на экран общее количество найденных строк "Go" во всех переданных URL, например:

```
$ echo -e 'https://golang.org\nhttps://golang.org' | go run main.go
Count for https://golang.org: 20
Count for https://golang.org: 20
Total: 40
```

Если дольше минуты нет нового ввода url приложение завершается.

Алгоритм:
* Считываение происходит в отдельной горутине.
* После считывания url кладется в канал, откуда его забирает другая горутина.
* В случае если количество обработчиков не превысило лимит (5) - начинается обработка url.
* Полученные данные кладутся в sync.Map (для исключения затирания одинаковых  url используется uuid).
* Ошибки при работе пишутся в отдельный канал, для считывания ошибок есть горутина, которая их выводит.
* При завершении приложение выводит результат работы (пример выше).

