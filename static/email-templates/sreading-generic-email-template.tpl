<!DOCTYPE html>
<html lang="zh">
  <title>{{ .Campaign.Subject }}</title>
  <body>
    <div style="min-width: 320px;max-width: 640px; margin: 0 auto">
      <!-- 头 -->
      <div
        class="header-color-css"
        style="
          padding: 10px 20px 7px 20px;
          display: flex;
          background-image: url({{ RootURL }}/uploads/sr-head-background.png);
          background-size: contain;
        "
      >
        <div class="app-loading-new">
          <img
            src="{{ RootURL }}/uploads/sreading-logo.svg"
            alt="SVG Image"
          />
        </div>
        <div style="padding-left: 10px; width: fit-content; text-align: left">
          <div
            style="
              font-size: 22px;
              line-height: 22px;
              font-weight: bold;
              text-shadow: 1px -1px 0 #ffffff, -1px -1px 0 #ffffff,
                -1px 1px 0 #ffffff, 1px 1px 0 #ffffff;
            "
          >
            SReading 从心所阅
          </div>
          <div
            style="
              margin-top: 6px;
              font-size: 21px;
              line-height: 22px;
              font-weight: bold;
              text-shadow: 1px -1px 0 #ffffff, -1px -1px 0 #ffffff,
                -1px 1px 0 #ffffff, 1px 1px 0 #ffffff;
            "
          >
            www.sreading.com
          </div>
        </div>
      </div>
      <!-- 内容 -->
      <div
        style="
          padding: 10px;
          background-color: #f3f3f8;
          margin-top: 10px;
          border-radius: 5px;
        "
      >
        <div
          style="padding: 10px; background-color: #ffffff; border-radius: 10px"
        >
          {{ template "content" . }}
        </div>
      </div>
      <!-- 脚 -->
      <div style="font-size: 10px; padding: 20px 10px">
        <div>SReading 人工智能驱动的中文学习平台，让您从心所阅，轻松高效</div>
        <div>
          在使用过程中发现任何问题，请您及时与我们联系。Email:
          <a href="mailto:support@sreading.org">support@sreading.org</a>
        </div>
        <div
          style="display: flex; justify-content: flex-end; align-items: center"
        >
          <a style="margin-right: 20px" href="mailto:support@sreading.org">
            联系我们</a
          ><a
            style="margin-right: 20px"
            href="https://www.sreading.org"
            target="_blank"
            >在线使用</a
          >
          <a
            href="https://apps.apple.com/us/app/sreading/id6446602590?itsct=apps_box_badge&amp;itscg=30200"
            style="display: inline-block; overflow: hidden"
            ><img
              src="https://tools.applemediaservices.com/api/badges/download-on-the-app-store/black/en-us?size=250x83&amp;releaseDate=1692316800"
              alt="Download on the App Store"
              style="border-radius: 2px; height: 27px"
          /></a>
          <a
            href="https://play.google.com/store/apps/details?id=org.sreading&pcampaignid=pcampaignidMKT-Other-global-all-co-prtnr-py-PartBadge-Mar2515-1"
            ><img
              style="height: 40px"
              alt="Get it on Google Play"
              src="https://play.google.com/intl/en_us/badges/static/images/badges/en_badge_web_generic.png"
          /></a>
        </div>
      </div>
    </div>
    <div>
      <div
        style="
          border-bottom: 1px #d8d8d8 solid;

          margin-top: 40px;
          height: 40px;
        "
      >
        <img
          style="float: left; margin-left: 20px"
          height="40px"
          src="{{ RootURL }}/uploads/penda.png"
        />
      </div>
      <div style="text-align: center; font-size: small">©2023 SReading</div>
    </div>
   <div class="footer" style="text-align: center;font-size: 12px;color: #888;">
           <p>
               <a href="{{ UnsubscribeURL }}?domain=sreading" style="color: #888;">{{ L.T "email.unsub" }}</a>
               &nbsp;&nbsp;
               <a href="{{ MessageURL }}" style="color: #888;">{{ L.T "email.viewInBrowser" }}</a>
           </p>
       </div>
  </body>
</html>
