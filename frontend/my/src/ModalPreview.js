import React from "react"
import { Modal } from "antd"
import * as cs from "./constants"

import { Spin } from "antd"

class ModalPreview extends React.PureComponent {
    makeForm(body) {
        let form = document.createElement("form")
        form.method = cs.MethodPost
        form.action = this.props.previewURL
        form.target = "preview-iframe"

        let input = document.createElement("input")
        input.type = "hidden"
        input.name = "body"
        input.value = body
        form.appendChild(input)
        document.body.appendChild(form)
        form.submit()
    }

    render () {
        return (
            <Modal visible={ true } title={ this.props.title }
                className="preview-modal"
                width="90%"
                height={ 900 }
                onCancel={ this.props.onCancel }
                onOk={ this.props.onCancel }>
                <div className="preview-iframe-container">
                        <Spin className="preview-iframe-spinner"></Spin>
                        <iframe key="xxxxxxxxx" onLoad={() => {
                            // If state is used to manage the spinner, it causes
                            // the iframe to re-render and reload everything.
                            // Hack the spinner away from the DOM directly instead.
                            let spin = document.querySelector(".preview-iframe-spinner")
                            if(spin) {
                                spin.parentNode.removeChild(spin)
                            }
                            // this.setState({ loading: false })
                        }} title={ this.props.title ? this.props.title : "Preview" }
                            name="preview-iframe"
                            id="preview-iframe"
                            className="preview-iframe"
                            ref={(o) => {
                                if(!o) {
                                    return
                                }

                                // When the DOM reference for the iframe is ready,
                                // see if there's a body to post with the form hack.
                                if(this.props.body !== undefined
                                    && this.props.body !== null) {
                                    this.makeForm(this.props.body)
                                } else {
                                    if(this.props.previewURL) {
                                        o.src = this.props.previewURL
                                    }
                                }
                            }}
                            src="about:blank">
                        </iframe>
                </div>
            </Modal>

        )
    }
}

export default ModalPreview
