import Uppy from '@uppy/core';
import DragDrop from '@uppy/drag-drop';
import StatusBar from '@uppy/status-bar';
import AwsS3 from '@uppy/aws-s3';

export default function (rootEl) {
    rootEl.querySelectorAll('[data-uppy]').forEach(el => {
        const presignURL = el.getAttribute('data-uppy-presign-url')

        const uppy = new Uppy({
            debug: true,
            autoProceed: true,
        });
        uppy.use(DragDrop, {
            target: el.querySelector('[data-uppy-drag-drop]'),
        });
        uppy.use(StatusBar, {
            target: el.querySelector('[data-uppy-status]'),
        });
        uppy.use(AwsS3, {
            getUploadParameters(file) {
                return fetch(presignURL, {
                    method: 'post',
                    headers: {
                        'Accept': 'application/json',
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        name: file.name,
                        content_type: file.type,
                    }),
                }).then((res) => {
                    return res.json()
                }).then((data) => {
                    return {
                        method: 'PUT',
                        url: data.url,
                        // headers: {
                        //     'Content-Type': 'multipart/form-data',
                        // },
                    }
                })
            },
        });
    });
}