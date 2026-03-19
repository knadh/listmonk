<!doctype html>
<html>
    <head>
        <title>{{ .Campaign.Subject }}</title>
        <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1">
        <base target="_blank">
        
        <link rel="preconnect" href="https://fonts.googleapis.com">
        <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
        <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600&family=Libre+Baskerville:wght@400;700&display=swap" rel="stylesheet">

        <style>
            /* 1. PERUSASETUKSET */
            body {
                background-color: #ffffff;
                margin: 0;
                padding: 0;
                color: #222222; 
                font-family: 'Inter', -apple-system, sans-serif;
                -webkit-font-smoothing: antialiased;
            }

            /* 2. PREHEADER (Uudet tyylit) */
            .preheader {
                max-width: 740px;
                margin: 0 auto;
                padding: 20px 30px 0 30px;
                text-align: right;
            }
            .preheader p {
                font-size: 12px !important;
                color: #999 !important;
                margin-bottom: 0 !important;
                line-height: 1 !important;
            }
            .preheader a {
                font-size: 12px !important;
                color: #999 !important;
                font-weight: 400 !important;
                text-decoration: none !important;
            }
            .preheader a:hover {
                text-decoration: underline !important;
                background-color: transparent !important;
                color: #111 !important;
            }

            /* 3. LEIPÄTEKSTI (21px) */
            p, li, div, td {
                font-size: 21px; 
                line-height: 1.6;
                margin-bottom: 24px;
                color: #222;
            }

            /* 4. OTSIKOT */
            h1, h2, h3 {
                font-family: 'Libre Baskerville', Georgia, serif;
                color: #111;
                margin-top: 0;
                line-height: 1.1;
                letter-spacing: -0.03em;
            }

            h1 {
                font-size: 58px !important;
                font-weight: 700;
                margin-bottom: 30px;
            }

            h2 {
                font-size: 34px !important;
                margin-top: 50px;
                margin-bottom: 20px;
                border-bottom: 2px solid #f0f0f0; 
                padding-bottom: 10px;
            }

            h3 {
                font-family: 'Inter', sans-serif;
                font-size: 16px !important;
                font-weight: 700;
                text-transform: uppercase;
                letter-spacing: 0.1em;
                color: #999;
                margin-top: 40px;
                margin-bottom: 10px;
            }

            /* 5. KUVAT */
            img {
                max-width: 100%;
                height: auto;
                display: block;
                border-radius: 4px; 
                margin-top: 40px;
                margin-bottom: 40px;
            }
            figcaption {
                font-size: 15px !important;
                color: #888;
                font-family: 'Inter', sans-serif;
                margin-top: -20px;
                margin-bottom: 30px;
            }

            /* 6. LINKIT */
            a {
                color: #111;
                text-decoration: underline;
                text-decoration-thickness: 1px;
                text-underline-offset: 3px;
                font-weight: 600;
            }
            a:hover {
                background-color: #f0f0f0;
                text-decoration: none;
            }

            /* 7. SOMMITTELU */
            hr {
                border: 0;
                border-top: 1px solid #e0e0e0;
                width: 15%;
                margin: 60px auto;
            }

            .button {
                display: inline-block;
                background-color: #111;
                color: #fff !important;
                text-decoration: none !important;
                padding: 18px 36px;
                border-radius: 4px;
                font-weight: 600;
                margin: 20px 0;
                font-size: 19px !important;
            }
            .button:hover {
                background-color: #444;
            }

            /* 8. LAINAUKSET */
            blockquote {
                font-family: 'Libre Baskerville', Georgia, serif;
                font-size: 28px !important;
                line-height: 1.4 !important;
                font-style: italic;
                margin: 40px 0;
                padding-left: 30px;
                border-left: 4px solid #111;
                color: #444;
            }

            /* MOBIILI */
            @media screen and (max-width: 600px) {
                p, li, div, td { font-size: 18px !important; }
                h1 { font-size: 38px !important; }
                h2 { font-size: 28px !important; }
                blockquote { font-size: 24px !important; padding-left: 20px !important; }
                .wrap { padding: 40px 20px !important; }
                .preheader { padding: 15px 20px 0 20px !important; }
            }
        </style>
    </head>
<body>
    <div class="preheader">
        <p>
            <a href="{{ UnsubscribeURL }}">{{ L.T "email.unsub" }}</a> &nbsp;&bull;&nbsp; 
            <a href="{{ MessageURL }}">{{ L.T "email.viewInBrowser" }}</a>
        </p>
    </div>

    <div class="wrap" style="max-width: 740px; margin: 0 auto; padding: 60px 30px;">
        {{ template "content" . }}
    </div>
    
    <div style="display:none; white-space:nowrap; font:15px courier; line-height:0;">{{ TrackView }}</div>
</body>
</html>
