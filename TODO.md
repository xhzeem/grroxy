## TODO
- Repeater: Update the content-length on change
- Raw Body Fix: -1 Content Length
- Pocketbase: Create realtime variable to limit the actions "create", "update", "delete" 
- Adding Icon:
      - wordpress
- Adding detections for:
      - cgi
      - .m3u8

- Changing this to admin middleware
```golang
admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
recordd, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

isGuest := admin == nil && recordd == nil

if isGuest {
      return c.String(http.StatusForbidden, "")
}
```

## Decisions to make
- `Backend` Host the images from the backend *(Maybe)*
- `Core` Should we save **same endpoint** giving **same data** multiple times? *(Ideally: NO)*
- `Data Size` Not saving unnecesarry raw data of images/videos/etc... *(Note: this will NOT impact the loading of images/content in the browser)*