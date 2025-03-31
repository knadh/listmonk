import { TEditorConfiguration } from '../../documents/editor/core';

const RESPOND_TO_MESSAGE: TEditorConfiguration = {
  root: {
    type: 'EmailLayout',
    data: {
      backdropColor: '#F0ECE5',
      canvasColor: '#F0ECE5',
      textColor: '#030303',
      fontFamily: 'MODERN_SERIF',
      childrenIds: [
        'block_HjX7RN8eDEz7BLBHSQCNgU',
        'block_Jf65r5cUAnEzBfxnHKGa5S',
        'block_WmPDNHDpyHkygqjHuqF7St',
        'block_4VCKUvRMo7EbuMdN1VsdRw',
        'block_4siwziT4BkewmN55PpXvEu',
        'block_S9Rg9F3bGcviRyfMpoU5e4',
      ],
    },
  },
  block_HjX7RN8eDEz7BLBHSQCNgU: {
    type: 'Image',
    data: {
      style: {
        padding: {
          top: 8,
          bottom: 24,
          left: 24,
          right: 24,
        },
      },
      props: {
        height: 32,
        url: 'https://d1iiu589g39o6c.cloudfront.net/live/platforms/platform_A9wwKSL6EV6orh6f/images/wptemplateimage_hW6RusynHUNTKoLm/boop.png',
        contentAlignment: 'middle',
      },
    },
  },
  block_Jf65r5cUAnEzBfxnHKGa5S: {
    type: 'Heading',
    data: {
      style: {
        color: null,
        backgroundColor: null,
        fontFamily: null,
        fontWeight: null,
        textAlign: null,
        padding: {
          top: 16,
          bottom: 0,
          left: 24,
          right: 24,
        },
      },
      props: {
        level: 'h2',
        text: `Respond to Anna's Inquiry`,
      },
    },
  },
  block_WmPDNHDpyHkygqjHuqF7St: {
    type: 'Text',
    data: {
      style: {
        color: null,
        backgroundColor: null,
        fontSize: 16,
        fontFamily: null,
        fontWeight: null,
        textAlign: null,
        padding: {
          top: 8,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'Dog boarding for Aug 1 - Aug 29.',
      },
    },
  },
  block_95nkowWyi4p2VBiA46Eizs: {
    type: 'Text',
    data: {
      style: {
        color: null,
        backgroundColor: '#faf9f9',
        fontSize: 21,
        fontFamily: null,
        fontWeight: null,
        textAlign: null,
        padding: {
          top: 24,
          bottom: 24,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'Any chance you can watch Emma again next month?',
      },
    },
  },
  block_4VCKUvRMo7EbuMdN1VsdRw: {
    type: 'Container',
    data: {
      style: {
        backgroundColor: null,
        borderColor: null,
        borderRadius: null,
        padding: {
          top: 16,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        childrenIds: ['block_95nkowWyi4p2VBiA46Eizs'],
      },
    },
  },
  block_4siwziT4BkewmN55PpXvEu: {
    type: 'Button',
    data: {
      style: {
        backgroundColor: null,
        fontSize: 16,
        fontFamily: null,
        fontWeight: null,
        textAlign: 'left',
        padding: {
          top: 24,
          bottom: 24,
          left: 24,
          right: 24,
        },
      },
      props: {
        buttonBackgroundColor: '#BE4F46',
        buttonTextColor: '#FFFFFF',
        size: 'large',
        buttonStyle: 'pill',
        text: 'Respond',
        url: 'http://example.usewaypoint.com/request/2334234',
        fullWidth: false,
      },
    },
  },
  block_S9Rg9F3bGcviRyfMpoU5e4: {
    type: 'Text',
    data: {
      style: {
        color: null,
        backgroundColor: null,
        fontSize: 16,
        fontFamily: null,
        fontWeight: 'normal',
        textAlign: 'left',
        padding: {
          top: 0,
          bottom: 16,
          left: 24,
          right: 24,
        },
      },
      props: {
        text: 'You need 2 more walks to become a Super Walker on Boop!',
      },
    },
  },
};

export default RESPOND_TO_MESSAGE;
