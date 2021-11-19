<mjml>
  <mj-head>
  	<mj-attributes>
      <mj-text font-family="Helvetica Neue, Segoe UI, Helvetica, sans-serif" color="#444" />
    </mj-attributes>
  </mj-head>
  <mj-body background-color="#F0F1F3">
    
    <mj-section>
      <mj-column>
        <mj-spacer height="50px" />
      </mj-column>
    </mj-section>

    <mj-section background-color="#fff" border-radius="5px">
      <mj-column>
        <mj-text font-size="15px" line-height="26px">
          {{ template "content" . }} 
        </mj-text>
      </mj-column>
    </mj-section>

    <mj-section>
      <mj-column>
        <mj-text font-size="12px" color="#888" align="center">
          {{ L.T "email.unsubHelp" }}
          <a href="{{ UnsubscribeURL }}" style="color: #888;">{{ L.T "email.unsub" }}</a>
        </mj-text>
        <mj-text font-size="12px" color="#888" align="center">
          Powered by <a href="https://listmonk.app" target="_blank" style="color: #888;">listmonk</a>
        </mj-text>
      </mj-column>
    </mj-section>

    <mj-section>
      <mj-column>
        <mj-raw>{{ TrackView }}</mj-raw>
      </mj-column>
    </mj-section>

  </mj-body>
</mjml>