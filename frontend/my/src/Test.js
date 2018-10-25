import React from "react";

import ReactQuill from "react-quill"
import "react-quill/dist/quill.snow.css"

const quillModules = {
    toolbar: {
		container: [
			[{"header": [1, 2, 3, false] }],
			["bold", "italic", "underline", "strike", "blockquote", "code"],
			[{ "color": [] }, { "background": [] }, { 'size': [] }],
			[{"list": "ordered"}, {"list": "bullet"}, {"indent": "-1"}, {"indent": "+1"}],
			[{"align": ""}, { "align": "center" }, { "align": "right" }, { "align": "justify" }],
			["link", "gallery"],
			["clean", "font"]
		],
		handlers: {
			"gallery": function() {
				
			}
		}
	}
}

class QuillEditor extends React.Component {
  componentDidMount() {
  }

  render() {
    return (
      <div>
          <ReactQuill
		  	modules={ quillModules }
		  	value="<h2>Welcome</h2>"
		  />
      </div>
    )
  }
}

export default QuillEditor;
