import type { App } from 'vue'
import {
  Button,
  Card,
  Checkbox,
  Col,
  Dialog,
  Dropdown,
  Form,
  FormItem,
  Icon,
  Input,
  Link,
  Loading,
  Menu,
  MenuItem,
  Option,
  Pagination,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  TabPanel,
  Tabs,
  Tag,
  Textarea,
  Upload,
} from 'tdesign-vue-next'

import 'tdesign-vue-next/es/button/style/css.mjs'
import 'tdesign-vue-next/es/card/style/css.mjs'
import 'tdesign-vue-next/es/checkbox/style/css.mjs'
import 'tdesign-vue-next/es/dialog/style/css.mjs'
import 'tdesign-vue-next/es/dropdown/style/css.mjs'
import 'tdesign-vue-next/es/form/style/css.mjs'
import 'tdesign-vue-next/es/grid/style/css.mjs'
import 'tdesign-vue-next/es/input/style/css.mjs'
import 'tdesign-vue-next/es/link/style/css.mjs'
import 'tdesign-vue-next/es/loading/style/css.mjs'
import 'tdesign-vue-next/es/menu/style/css.mjs'
import 'tdesign-vue-next/es/message/style/css.mjs'
import 'tdesign-vue-next/es/pagination/style/css.mjs'
import 'tdesign-vue-next/es/popconfirm/style/css.mjs'
import 'tdesign-vue-next/es/select/style/css.mjs'
import 'tdesign-vue-next/es/space/style/css.mjs'
import 'tdesign-vue-next/es/table/style/css.mjs'
import 'tdesign-vue-next/es/tabs/style/css.mjs'
import 'tdesign-vue-next/es/tag/style/css.mjs'
import 'tdesign-vue-next/es/textarea/style/css.mjs'
import 'tdesign-vue-next/es/upload/style/css.mjs'

const components = [
  Button,
  Card,
  Checkbox,
  Col,
  Dialog,
  Dropdown,
  Form,
  FormItem,
  Icon,
  Input,
  Link,
  Loading,
  Menu,
  MenuItem,
  Option,
  Pagination,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  TabPanel,
  Tabs,
  Tag,
  Textarea,
  Upload,
]

export function registerTDesign(app: App) {
  components.forEach(component => {
    app.use(component)
  })
}
