<mjml>
  <mj-head>
    <mj-title>{{ .Campaign.Subject }}</mj-title>
    <mj-preview>{{ .Campaign.Subject }}</mj-preview>
  </mj-head>
  <mj-body background-color="#F0F1F3">
    <!-- Spacer -->
    <mj-section padding="30px 0">
      <mj-column>
        <mj-text>&nbsp;</mj-text>
      </mj-column>
    </mj-section>
    
    <!-- Main Content Wrapper -->
    <mj-section background-color="#fff" border-radius="5px" padding="30px">
      <mj-column>
        {{ template "content" . }}
      </mj-column>
    </mj-section>
    
    <!-- Footer -->
    <mj-section padding="20px 0">
      <mj-column>
        <mj-text align="center" font-size="12px" color="#888">
          <a href="{{ UnsubscribeURL }}" style="color: #888; margin-right: 5px;">{{ L.T "email.unsub" }}</a>
          &nbsp;&nbsp;
          <a href="{{ MessageURL }}" style="color: #888; margin-right: 5px;">{{ L.T "email.viewInBrowser" }}</a>
        </mj-text>
      </mj-column>
    </mj-section>
    
    <!-- Bottom Spacer with Tracking -->
    <mj-section padding="30px 0">
      <mj-column>
        <mj-raw>
          &nbsp;{{ TrackView }}
        </mj-raw>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>
