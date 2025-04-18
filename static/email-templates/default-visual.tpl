<!DOCTYPE html>
<html>
   <body>
      <div style="background-color:#F5F5F5;color:#262626;font-family:&quot;Helvetica Neue&quot;, &quot;Arial Nova&quot;, &quot;Nimbus Sans&quot;, Arial, sans-serif;font-size:16px;font-weight:400;letter-spacing:0.15008px;line-height:1.5;margin:0;padding:32px 0;min-height:100%;width:100%">
         <table align="center" width="100%" style="margin:0 auto;max-width:600px;background-color:#FFFFFF" role="presentation" cellSpacing="0" cellPadding="0" border="0">
            <tbody>
               <tr style="width:100%">
                  <td>
                     <h3 style="font-weight:bold;margin:0;font-size:20px;padding:16px 24px 16px 24px">Hello {{ .Subscriber.Name }}</h3>
                     <div style="font-weight:normal;padding:16px 24px 16px 24px">
                        <p>This is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and this block of text is in Markdown.</p>
                        <p>Here is a <a href="https://listmonk.app@TrackLink" target="_blank">tracked link</a>.</p>
                        <p>Use the link icon in the editor toolbar or when writing raw HTML or Markdown, simply suffix @TrackLink to the end of a URL to turn it into a tracking link. Example:</p>
                        <p><a href="https:/‌/listmonk.app@TrackLink"></a></p>
                        <p>For help, refer to the <a href="https://listmonk.app/docs" target="_blank">documentation</a>.</p>
                     </div>
                     <div style="padding:16px 0px 16px 0px">
                        <hr style="width:100%;border:none;border-top:1px solid #CCCCCC;margin:0"/>
                     </div>
                     <div style="padding:16px 24px 16px 24px">
                        <a href="https://listmonk.app" style="color:#FFFFFF;font-size:16px;font-weight:bold;background-color:#0055d4;border-radius:4px;display:inline-block;padding:12px 20px;text-decoration:none" target="_blank">
                           <span>
                              <!--[if mso]><i style="letter-spacing: 20px;mso-font-width:-100%;mso-text-raise:30" hidden>&nbsp;</i><![endif]-->
                           </span>
                           <span>This is a button</span>
                           <span>
                              <!--[if mso]><i style="letter-spacing: 20px;mso-font-width:-100%" hidden>&nbsp;</i><![endif]-->
                           </span>
                        </a>
                     </div>
                  </td>
               </tr>
            </tbody>
         </table>
      </div>
   </body>
</html>
