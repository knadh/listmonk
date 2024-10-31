import { TEditorConfiguration } from '../../documents/editor/core';

const RESET_PASSWORD: TEditorConfiguration = {
  root: {
    type: 'EmailLayout',
    data: {
      backdropColor: '#F2F5F7',
      canvasColor: '#FFFFFF',
      textColor: '#242424',
      fontFamily: 'MODERN_SANS',
      childrenIds: [
        'block_3gpSGmkgL4nWSBQjWCjK2z',
        'block_BjpQ7DGTtvaEuYRMd7VE7w',
        'block_xyg4GWmgGbJJEDRQc76bC',
        'block_76VptLCZ47t3EkAarUufEJ',
        'block_Gtk3kDYwsJqEmQf2XGWPRc',
        'block_LACDCzUS2bsvEbmnq1KHuW',
      ],
    },
  },
  block_3gpSGmkgL4nWSBQjWCjK2z: {
    type: 'Image',
    data: {
      style: {
        padding: {
          top: 24,
          bottom: 8,
          right: 24,
          left: 24,
        },
        backgroundColor: null,
        textAlign: 'left',
      },
      props: {
        height: 24,
        url: 'https://d1iiu589g39o6c.cloudfront.net/live/platforms/platform_A9wwKSL6EV6orh6f/images/wptemplateimage_Xh1R23U9ziyct9nd/codoc.png',
        alt: '',
        linkHref: null,
        contentAlignment: 'middle',
      },
    },
  },
  block_BjpQ7DGTtvaEuYRMd7VE7w: {
    type: 'Heading',
    data: {
      style: {
        color: null,
        backgroundColor: null,
        fontFamily: null,
        fontWeight: 'bold',
        textAlign: 'left',
        padding: {
          top: 32,
          bottom: 0,
          left: 24,
          right: 24,
        },
      },
      props: {
        level: 'h3',
        text: 'Reset your password?',
      },
    },
  },
  block_xyg4GWmgGbJJEDRQc76bC: {
    type: 'Text',
    data: {
      style: {
        color: '#474849',
        backgroundColor: null,
        fontSize: 14,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'left',
        padding: {
          top: 8,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: `If you didn't request a reset, don't worry. You can safely ignore this email.`,
      },
    },
  },
  block_76VptLCZ47t3EkAarUufEJ: {
    type: 'Button',
    data: {
      style: {
        backgroundColor: null,
        fontSize: 14,
        fontFamily: null,
        fontWeight: 'bold',
        textAlign: 'left',
        padding: {
          top: 12,
          bottom: 32,
          right: 24,
          left: 24,
        },
      },
      props: {
        buttonBackgroundColor: '#0068FF',
        buttonStyle: 'rectangle',
        buttonTextColor: '#FFFFFF',
        fullWidth: false,
        size: 'medium',
        text: 'Change my password',
        url: 'https://example.usewaypoint.com/reset_password?token=02938409809w8r09a83wr098aw0',
      },
    },
  },
  block_Gtk3kDYwsJqEmQf2XGWPRc: {
    type: 'Divider',
    data: {
      style: {
        backgroundColor: null,
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        lineHeight: 1,
        lineColor: '#EEEEEE',
      },
    },
  },
  block_LACDCzUS2bsvEbmnq1KHuW: {
    type: 'Text',
    data: {
      style: {
        color: '#474849',
        backgroundColor: null,
        fontSize: 12,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'left',
        padding: {
          top: 4,
          bottom: 24,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'Need help? Just reply to this email to contact support.',
      },
    },
  },
};

export default RESET_PASSWORD;
