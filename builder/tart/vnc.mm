#import <stdio.h>

#import <CoreGraphics/CoreGraphics.h>
#import <Vision/Vision.h>

struct Rectangle {
    double MinX;
    double MinY;
    double MaxX;
    double MaxY;
};

bool recognizeTextInFramebuffer(
  const char *text,
  void *framebuffer,
  int width,
  int height,
  struct Rectangle *output
) {
    @autoreleasepool {
        // Prepare regular expression for needle
        NSError *error = nil;
        NSRegularExpression *regex = [NSRegularExpression regularExpressionWithPattern:
            [NSString stringWithUTF8String:text] options:NSRegularExpressionCaseInsensitive
                error:&error];
        if (error) {
            fprintf(stderr, "⚠️ Failed to create search string regex: %s\n",
                error.localizedDescription.UTF8String);
            return false;
        }

        // Create CGImage wrapper around framebuffer pixel data
        CGDataProviderRef provider = CGDataProviderCreateWithData(
            NULL, framebuffer, width * height * 4, NULL);
        CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
        CGImageRef image = CGImageCreate(width, height, 8, 32, width * 4,
            colorSpace, (CGBitmapInfo)kCGImageAlphaPremultipliedLast, provider,
            NULL, false, kCGRenderingIntentDefault);
        CGColorSpaceRelease(colorSpace);
        CGDataProviderRelease(provider);

        // Recognize text in the framebuffer
        VNRecognizeTextRequest *textRecognizer = [[VNRecognizeTextRequest alloc] init];
        textRecognizer.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
        VNImageRequestHandler *imageRequest = [[VNImageRequestHandler alloc]
            initWithCGImage:image options:@{}];
        CGImageRelease(image);

        BOOL ret = [imageRequest performRequests:@[textRecognizer] error:&error];
        if (error || !ret) {
            fprintf(stderr, "⚠️ Failed to perform image recognition request: %s\n",
                error.localizedDescription.UTF8String);
            return false;
        }

        // Then search for the needle
        for (VNRecognizedTextObservation *observation in textRecognizer.results) {
            for (VNRecognizedText *candidate in [observation topCandidates:1]) {
                fprintf(stderr, "💬 Observed '%s' with confidence %f\n",
                    candidate.string.UTF8String, candidate.confidence);
                NSRange range = NSMakeRange(0, candidate.string.length);
                if ([regex matchesInString:candidate.string options:0 range:range].count > 0) {
                    VNRectangleObservation *rect = [candidate boundingBoxForRange:range error:&error];
                    if (error || !rect) {
                      fprintf(stderr, "⚠️ Failed to perform image recognition request: %s\n",
                          error.localizedDescription.UTF8String);
                      return false;
                    }

                    output->MinX = fmin(
                      fmin(rect.topLeft.x, rect.topRight.x),
                      fmin(rect.bottomLeft.x, rect.bottomRight.x)
                    );
                    output->MaxX = fmax(
                      fmax(rect.topLeft.x, rect.topRight.x),
                      fmax(rect.bottomLeft.x, rect.bottomRight.x)
                    );

                    // Flip the axis, because:
                    //
                    // >OCR results are also reported in normalized coordinates, but with the origin at lower left.
                    //
                    // https://rethunk.medium.com/coordinate-transforms-in-ios-using-swift-part-1-the-l-triangle-c8204177a7e2
                    output->MinY = fmin(
                      fmin(1.0 - rect.topLeft.y, 1.0 - rect.topRight.y),
                      fmin(1.0 - rect.bottomLeft.y, 1.0 - rect.bottomRight.y)
                    );
                    output->MaxY = fmax(
                      fmax(1.0 - rect.topLeft.y, 1.0 - rect.topRight.y),
                      fmax(1.0 - rect.bottomLeft.y, 1.0 - rect.bottomRight.y)
                    );

                    return true;
                }
            }
        }
    }

    return false;
}
