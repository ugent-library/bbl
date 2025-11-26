# Documentation

## Query language

```
(status=suggestion AND (kind=book OR kind=journal_article)) OR kind=dissertation AND status=draft
```

* AND has precedence over OR

    ```
    kind=dissertation and status=suggestion or status=draft
    ```

    Is the same as:

    ```
    (kind=dissertation and status=suggestion) or status=draft
    ```

* boolean operators are case insensitive

    ```
    kind = "dissertation" and status = "suggestion"
    kind = "dissertation" AND status = "suggestion"
    ```

* string quoting is optional if there is no ambiguity

    ```
    kind = "dissertation" and status = "suggestion"
    kind = dissertation and status = suggestion
    ```

* values can be seperated by a "|"

    ```
    status=suggestion|draft
    ```

    Is the same as:

    ```
    status=suggestion or status=draft
    ```
