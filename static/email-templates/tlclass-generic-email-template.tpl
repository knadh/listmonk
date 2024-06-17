<!DOCTYPE html>
<html lang="zh">
  <title>{{ .Campaign.Subject }}</title>
  <body>
    <div style="max-width: 560px; margin: auto; margin-bottom: 30px">
      <!-- 图标 -->
      <div style="height: 40px">
        <img
          height="40px"
          src="{{ RootURL }}/uploads/tlc-logo-title.png"
        />
      </div>
      <!-- 同风万里，乐学致远 -->
      <div>
        <img
          style="float: right"
          height="40px"
          src="{{ RootURL }}/uploads/loong.png"
        />
      </div>
      <div style="margin-top: 30px">
        <div
          style="
            font-size: x-large;
            background-color: #ea4335;
            padding: 8px;
            text-align: center;
            color: #ffffff;
          "
        >
          同风万里，乐学致远
        </div>
      </div>
      <!-- 内容 -->
      <div style="margin-top: 10px; padding: 2px; border: 2px solid #ea4335">
        <div style="border: 1px solid #ea4335; padding: 15px">
          {{ template "content" . }}
        </div>
      </div>
      <!-- 脚 -->
      <div style="font-size: small; line-height: 25px; margin-top: 20px">
        <div>
          如果需要帮助，请联系<a href="mailto:registrar@tonglec.org"
            >registrar@tonglec.org</a
          >
          与课程顾问联系，我们会尽快回复您。
        </div>
        <div>同乐中文，致力于海外高质量中文教育，让孩子走得更高，更远！</div>
      </div>
    </div>
    <div
      style="
        border-top: 1px #ffd123 solid;
        text-align: center;
        font-size: small;
      "
    >
      版权所有©同乐文化科技有限公司 | 地址: 14200 SE 13th Pl, Bellevue, WA 98007
    </div>
    <div class="footer" style="text-align: center;font-size: 12px;color: #888;">
            <p>
                <a href="{{ UnsubscribeURL }}?domain=tlclass" style="color: #888;">{{ L.T "email.unsub" }}</a>
                &nbsp;&nbsp;
                <a href="{{ MessageURL }}" style="color: #888;">{{ L.T "email.viewInBrowser" }}</a>
            </p>
        </div>
  </body>
</html>
